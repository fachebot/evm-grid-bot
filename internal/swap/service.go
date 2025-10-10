package swap

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strings"

	"github.com/fachebot/evm-grid-bot/internal/dexagg/relaylink"
	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/settings"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/utils/evm"

	"github.com/ethereum/go-ethereum/crypto"
)

type SwapService struct {
	svcCtx   *svc.ServiceContext
	userId   int64
	prv      *ecdsa.PrivateKey
	settings *ent.Settings
}

func NewSwapService(svcCtx *svc.ServiceContext, userId int64) *SwapService {
	return &SwapService{svcCtx: svcCtx, userId: userId}
}

func (s *SwapService) Quote(ctx context.Context, inputToken, outputToken string, amount *big.Int, exit ...bool) (SwapTransaction, error) {
	userWallet, err := s.getUserWallet(ctx)
	if err != nil {
		return nil, err
	}

	userSettings, err := s.getUserSettings(ctx)
	if err != nil {
		return nil, err
	}

	slippageBps := int(userSettings.SlippageBps)
	if strings.EqualFold(outputToken, s.svcCtx.Config.Chain.StablecoinCA) && userSettings.SellSlippageBps != nil {
		slippageBps = int(*userSettings.SellSlippageBps)
		if len(exit) > 0 && exit[0] && userSettings.ExitSlippageBps != nil {
			// 如果是清仓交易，使用清仓滑点
			slippageBps = int(*userSettings.ExitSlippageBps)
		}
	}

	switch userSettings.DexAggregator {
	case settings.DexAggregatorRelay:
		user, err := evm.GetAddress(userWallet)
		if err != nil {
			return nil, err
		}

		var enableInfiniteApproval bool
		if userSettings.EnableInfiniteApproval != nil && *userSettings.EnableInfiniteApproval {
			enableInfiniteApproval = true
		}

		relaylinkClient := relaylink.NewRelaylinkClient(s.svcCtx.TransportProxy)
		quoteResponse, err := relaylinkClient.Quote(ctx, s.svcCtx.Config.Chain.Id, user.Hex(), inputToken, outputToken, amount, slippageBps, enableInfiniteApproval)
		if err != nil {
			return nil, err
		}
		return NewRelaySwapTransaction(s, quoteResponse, user.Hex()), nil
	default:
		return nil, errors.New("unsupported aggregator")
	}

}

func (s *SwapService) getUserWallet(ctx context.Context) (*ecdsa.PrivateKey, error) {
	if s.prv != nil {
		return s.prv, nil
	}

	w, err := s.svcCtx.WalletModel.FindByUserId(ctx, s.userId)
	if err != nil {
		logger.Errorf("[SwapService] 查询用户钱包失败, userId: %d, %v", s.userId, err)
		return nil, err
	}

	pk, err := s.svcCtx.HashEncoder.Decryption(w.PrivateKey)
	if err != nil {
		logger.Errorf("[SwapService] 解密用户私钥失败, userId: %d, %v", s.userId, err)
		return nil, err
	}

	prv, err := crypto.HexToECDSA(pk)
	if err != nil {
		logger.Errorf("[SwapService] 解析用户私钥失败, userId: %d, %v", s.userId, err)
		return nil, err
	}

	s.prv = prv

	return prv, nil
}

func (s *SwapService) getUserSettings(ctx context.Context) (*ent.Settings, error) {
	if s.settings != nil {
		return s.settings, nil
	}

	v, err := s.svcCtx.SettingsModel.FindByUserId(ctx, s.userId)
	if err == nil {
		s.settings = v
		return v, nil
	}

	if !ent.IsNotFound(err) {
		logger.Errorf("[SwapService] 查询用户设置失败, userId: %d, %v", s.userId, err)
		return nil, err
	}

	c := s.svcCtx.Config.Chain
	ret := ent.Settings{
		UserId:        s.userId,
		SlippageBps:   c.SlippageBps,
		DexAggregator: settings.DexAggregator(c.DexAggregator),
	}

	s.settings = &ret

	return &ret, nil
}
