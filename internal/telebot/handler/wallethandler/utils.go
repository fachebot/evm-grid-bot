package wallethandler

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/utils"
	"github.com/fachebot/evm-grid-bot/internal/utils/evm"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GetUserWallet(ctx context.Context, svcCtx *svc.ServiceContext, userId int64) (*ent.Wallet, error) {
	w, err := svcCtx.WalletModel.FindByUserId(ctx, userId)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, err
		}

		privateKey, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}

		privateKeyBytes := crypto.FromECDSA(privateKey)
		pk, err := svcCtx.HashEncoder.Encryption(hexutil.Encode(privateKeyBytes)[2:])
		if err != nil {
			return nil, err
		}

		publicKey := privateKey.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		}

		args := ent.Wallet{
			UserId:     userId,
			Account:    crypto.PubkeyToAddress(*publicKeyECDSA).Hex(),
			PrivateKey: pk,
		}
		w, err = svcCtx.WalletModel.Save(ctx, args)
		if err != nil {
			return nil, err
		}
	}

	return w, nil
}

func DisplayWalletMenu(ctx context.Context, svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI, userId int64, update tgbotapi.Update) error {
	// 确保生成账户
	w, err := GetUserWallet(ctx, svcCtx, userId)
	if err != nil {
		return err
	}

	// 查询账户余额
	balance, err := evm.GetBalance(ctx, svcCtx.EthClient, w.Account)
	if err != nil {
		balance = big.NewInt(0)
	}

	// 获取元数据
	var decimals uint8
	tokenmeta, err := svcCtx.TokenMetaCache.GetTokenMeta(ctx, svcCtx.Config.Chain.StablecoinCA)
	if err == nil {
		decimals = tokenmeta.Decimals
	}

	// 查询USD余额
	usdBalance, err := evm.GetTokenBalance(ctx, svcCtx.EthClient, svcCtx.Config.Chain.StablecoinCA, w.Account)
	if err != nil {
		usdBalance = big.NewInt(0)
	}

	// 回复钱包菜单
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ 返回", "/home"),
			tgbotapi.NewInlineKeyboardButtonData("刷新余额", WalletHomeHandler{}.FormatPath()),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚠️ 导出钱包私钥", KeyExportHandler{}.FormatPath(w.Account)),
		),
	)

	chainId := svcCtx.Config.Chain.Id
	currency := svcCtx.Config.Chain.NativeCurrency.Symbol
	stablecoinSymbol := svcCtx.Config.Chain.StablecoinSymbol
	text := fmt.Sprintf("%s 网格机器人 | 钱包管理\n\n💳 我的钱包:\n`%s`\n\n💰 %s余额: `%s`\n💰 %s余额: `%s`",
		utils.GetNetworkName(chainId), w.Account, currency, evm.ParseETH(balance).Truncate(5), stablecoinSymbol, evm.ParseUnits(usdBalance, decimals).Truncate(5))

	text = text + fmt.Sprintf("\n\n[OKX](%s) | [GMGN](%s) | [BlockExplorer](%s)",
		utils.GetOkxAccountLink(chainId, w.Account), utils.GetGmgnAccountLink(chainId, w.Account), utils.GetBlockExplorerAccountLink(chainId, w.Account))
	_, err = utils.ReplyMessage(botApi, update, text, markup)
	return err
}
