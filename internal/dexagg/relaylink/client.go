package relaylink

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/fachebot/evm-grid-bot/internal/dexagg"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/utils/evm"

	"github.com/carlmjohnson/requests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var (
	ETH = common.HexToAddress("0x0000000000000000000000000000000000000000")
)

// 链列表缓存配置
const (
	chainsCacheTTL       = 1 * time.Hour // 链列表缓存有效期
	chainsFetchRetries   = 3             // 链列表拉取重试次数
	chainsFetchRetryWait = time.Second   // 重试间隔
)

// globalChainsCache 全局链列表缓存
// 避免每次报价都重新拉取全量链列表(62条链的大JSON), 刷新失败时回退到上次快照
var globalChainsCache = struct {
	mutex     sync.Mutex
	chains    map[int64]Chain
	fetchedAt time.Time
}{}

type RelaylinkClient struct {
	transportProxy *http.Transport
}

func NewRelaylinkClient(transportProxy *http.Transport) *RelaylinkClient {
	client := &RelaylinkClient{
		transportProxy: transportProxy,
	}
	return client
}

func (client *RelaylinkClient) GetChains(ctx context.Context) ([]Chain, error) {
	httpClient := new(http.Client)
	if client.transportProxy != nil {
		httpClient.Transport = client.transportProxy
	}

	var chains Chains
	err := requests.URL("https://api.relay.link/chains").
		Client(httpClient).
		ToJSON(&chains).
		Fetch(ctx)

	return chains.Chains, err
}

// getChains 获取链列表(带全局缓存)
// 缓存有效期内直接返回; 过期或为空时刷新; 刷新失败回退到上次成功快照
func (client *RelaylinkClient) getChains(ctx context.Context) (map[int64]Chain, error) {
	globalChainsCache.mutex.Lock()
	defer globalChainsCache.mutex.Unlock()

	if len(globalChainsCache.chains) > 0 && time.Since(globalChainsCache.fetchedAt) < chainsCacheTTL {
		return globalChainsCache.chains, nil
	}

	chains, err := client.fetchChainsWithRetry(ctx)
	if err != nil {
		if len(globalChainsCache.chains) > 0 {
			logger.Warnf("[RelaylinkClient] 刷新链列表失败, 使用上次缓存, %v", err)
			return globalChainsCache.chains, nil
		}
		return nil, err
	}

	globalChainsCache.chains = chains
	globalChainsCache.fetchedAt = time.Now()
	return chains, nil
}

// fetchChainsWithRetry 拉取链列表并自动重试
func (client *RelaylinkClient) fetchChainsWithRetry(ctx context.Context) (map[int64]Chain, error) {
	var lastErr error
	for i := 0; i < chainsFetchRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(chainsFetchRetryWait):
			}
		}

		chains, err := client.fetchChains(ctx)
		if err == nil {
			return chains, nil
		}
		lastErr = err
		logger.Warnf("[RelaylinkClient] 拉取链列表失败(第%d次), %v", i+1, err)
	}

	return nil, lastErr
}

// fetchChains 拉取全量链列表并建立链ID索引
func (client *RelaylinkClient) fetchChains(ctx context.Context) (map[int64]Chain, error) {
	chains, err := client.GetChains(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch relay chains: %w", err)
	}

	result := make(map[int64]Chain, len(chains))
	for _, chain := range chains {
		result[chain.ID] = chain
	}
	return result, nil
}

func (client *RelaylinkClient) GetChainsByID(ctx context.Context, chainId int64) (Chain, error) {
	chains, err := client.getChains(ctx)
	if err != nil {
		return Chain{}, err
	}

	chain, ok := chains[chainId]
	if !ok {
		return Chain{}, fmt.Errorf("unsupported chain: %d", chainId)
	}
	return chain, nil
}

func (client *RelaylinkClient) Quote(
	ctx context.Context,
	chainId int64,
	user,
	inputToken,
	outputToken string,
	amount *big.Int,
	slippageBps int,
	enableInfiniteApproval bool,
) (*QuoteResponse, error) {
	chain, err := client.GetChainsByID(ctx, chainId)
	if err != nil {
		return nil, err
	}

	params := map[string]any{
		"user":                user,
		"originChainId":       chain.ID,
		"destinationChainId":  chain.ID,
		"originCurrency":      inputToken,
		"destinationCurrency": outputToken,
		"amount":              amount.String(),
		"tradeType":           "EXACT_INPUT",
		"slippageTolerance":   slippageBps,
	}

	httpClient := new(http.Client)
	if client.transportProxy != nil {
		httpClient.Transport = client.transportProxy
	}

	var errRes *ErrorResponse
	var response QuoteResponse
	err = requests.URL("https://api.relay.link/quote").
		Method(http.MethodPost).
		Client(httpClient).
		BodyJSON(params).
		ErrorJSON(&errRes).
		ToJSON(&response).
		Fetch(ctx)
	if err != nil {
		if errRes != nil {
			return nil, errRes
		}
		return nil, err
	}

	if enableInfiniteApproval {
		for _, step := range response.Steps {
			for idx, item := range step.Items {
				if step.ID != "approve" {
					continue
				}

				data, err := hexutil.Decode(item.Data.EvmData)
				if err != nil {
					return nil, err
				}

				spender, _, err := evm.DecodeERC20ApproveInput(data)
				if err != nil {
					continue
				}

				input, err := evm.EncodeERC20ApproveInput(spender, evm.MaxUint256)
				if err == nil {
					step.Items[idx].Data.EvmData = hexutil.Encode(input)
				}
			}

		}
	}

	return &response, nil
}

func (client *RelaylinkClient) SendSwapTransaction(ctx context.Context, svcCtx *svc.ServiceContext, prv *ecdsa.PrivateKey, swapResponse *QuoteResponse) (string, uint64, error) {
	account, err := evm.GetAddress(prv)
	if err != nil {
		return "", 0, err
	}

	// 检查余额
	ethBal, err := evm.GetBalance(ctx, svcCtx.EthClient, account.Hex())
	if err != nil {
		return "", 0, err
	}
	if ethBal.Cmp(swapResponse.Fees.Gas.Amount.BigInt()) <= 0 {
		return "", 0, dexagg.ErrInsufficientBalance
	}

	inTokenBal, err := evm.GetTokenBalance(ctx, svcCtx.EthClient, swapResponse.Details.CurrencyIn.Currency.Address, account.Hex())
	if err != nil {
		return "", 0, err
	}
	if inTokenBal.Cmp(swapResponse.Details.CurrencyIn.Amount.BigInt()) < 0 {
		return "", 0, dexagg.ErrInsufficientBalance
	}

	// 发送交易
	var lastTxHash string
	var lastTxNonce uint64
	chainId := uint64(svcCtx.Config.Chain.Id)
	for _, step := range swapResponse.Steps {
		for _, item := range step.Items {
			data, err := hexutil.Decode(item.Data.EvmData)
			if err != nil {
				return "", 0, err
			}

			err = svcCtx.NonceManager.Request(ctx, account, func(ctx context.Context, nonce uint64) (hash string, err error) {

				dynamicFeeTx := ethtypes.DynamicFeeTx{
					ChainID:   big.NewInt(0).SetUint64(chainId),
					Nonce:     nonce,
					GasTipCap: item.Data.EvmMaxPriorityFeePerGas.BigInt(),
					GasFeeCap: item.Data.EvmMaxFeePerGas.BigInt(),
					Gas:       item.Data.EvmGas.BigInt().Uint64(),
					To:        &item.Data.EvmTo,
					Value:     item.Data.EvmValue.BigInt(),
					Data:      data,
				}

				tx := ethtypes.NewTx(&dynamicFeeTx)
				signedTx, err := ethtypes.SignTx(tx, ethtypes.NewLondonSigner(big.NewInt(0).SetUint64(chainId)), prv)
				if err != nil {
					return "", err
				}

				err = svcCtx.EthClient.SendTransaction(ctx, signedTx)
				if err != nil {
					return "", err
				}

				lastTxNonce = nonce
				lastTxHash = signedTx.Hash().Hex()

				return signedTx.Hash().Hex(), nil
			})

			if err != nil {
				return "", 0, err
			}
		}
	}

	return lastTxHash, lastTxNonce, nil
}
