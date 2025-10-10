package relaylink

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"

	"github.com/fachebot/evm-grid-bot/internal/dexagg"
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

type RelaylinkClient struct {
	transportProxy   *http.Transport
	chainsCache      map[int64]Chain
	chainsCacheMutex sync.Mutex
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

func (client *RelaylinkClient) GetChainsByID(ctx context.Context, chainId int64) (Chain, bool) {
	client.chainsCacheMutex.Lock()
	defer client.chainsCacheMutex.Unlock()

	if client.chainsCache == nil {
		chains, err := client.GetChains(ctx)
		if err != nil {
			return Chain{}, false
		}

		client.chainsCache = make(map[int64]Chain)
		for _, chain := range chains {
			client.chainsCache[chain.ID] = chain
		}
	}

	chain, ok := client.chainsCache[chainId]
	return chain, ok
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
	chain, ok := client.GetChainsByID(ctx, chainId)
	if !ok {
		return nil, errors.New("unsupported chain")
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

	d, err := json.Marshal(params)
	if err == nil {
		fmt.Println(string(d))
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
