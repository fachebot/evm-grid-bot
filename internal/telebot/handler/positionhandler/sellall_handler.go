package positionhandler

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/fachebot/evm-grid-bot/internal/cache"
	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/order"
	"github.com/fachebot/evm-grid-bot/internal/ent/strategy"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/swap"
	"github.com/fachebot/evm-grid-bot/internal/telebot/handler/wallethandler"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/evm-grid-bot/internal/utils"
	"github.com/fachebot/evm-grid-bot/internal/utils/evm"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shopspring/decimal"
)

type SellAllHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewSellAllHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *SellAllHandler {
	return &SellAllHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h SellAllHandler) FormatPath() string {
	return "/position/sellall"
}

func (h *SellAllHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/position/sellall", h.handle)
}

func (h *SellAllHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID

		// 要求输入合约地址
		text := "请输入需要清仓的代币合约地址:"
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[SellAllHandler] 发送消息失败, %v", err)
		}

		route := cache.RouteInfo{Path: h.FormatPath(), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 获取用户钱包
	w, err := wallethandler.GetUserWallet(ctx, h.svcCtx, userId)
	if err != nil {
		return err
	}

	if update.Message != nil {
		chatId := update.Message.Chat.ID

		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		token := update.Message.Text
		if strings.EqualFold(token, h.svcCtx.Config.Chain.StablecoinCA) {
			text := fmt.Sprintf("❌ 不能清仓 %s 代币", h.svcCtx.Config.Chain.StablecoinSymbol)
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)
			return nil
		}

		// 策略是否正在运行
		s, err := h.svcCtx.StrategyModel.FindByUserIdToken(ctx, userId, token)
		if err == nil {
			if s.Status == strategy.StatusActive {
				utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 清仓前请手动停止正在运行的策略", 1)
				return nil
			}
		} else if !ent.IsNotFound(err) {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 服务器内部错误, 请稍后再试", 1)
			return nil
		}

		// 获取元数据
		tokenmeta, err := h.svcCtx.TokenMetaCache.GetTokenMeta(ctx, token)
		if err != nil {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 查询元数据失败, 请检查后再试", 1)
			return nil
		}

		// 查询代币余额
		balance, err := evm.GetTokenBalance(ctx, h.svcCtx.EthClient, token, w.Account)
		if err != nil {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 查询代币余额失败, 请检查后再试", 1)
			return nil
		}
		uiBalance := evm.ParseUnits(balance, tokenmeta.Decimals)
		if uiBalance.LessThanOrEqual(decimal.Zero) {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "🟢 此代币余额为零, 无需清仓", 1)
			return nil
		}

		h.handleSellAll(ctx, userId, chatId, update.Message.Text, tokenmeta.Symbol, tokenmeta.Decimals, balance)
	}

	return nil
}

func (h *SellAllHandler) handleSellAll(ctx context.Context, userId int64, chatId int64, token, symbol string, decimals uint8, amount *big.Int) {
	uiAmount := evm.ParseUnits(amount, decimals)
	utils.SendMessageAndDelayDeletion(h.botApi, chatId, fmt.Sprintf("📊 代币持仓: %s 枚 | ⚡️ 清仓中...", uiAmount), 1)

	// 获取报价
	swapService := swap.NewSwapService(h.svcCtx, userId)
	tx, err := swapService.Quote(ctx, token, h.svcCtx.Config.Chain.StablecoinCA, amount, true)
	if err != nil {
		logger.Errorf("[SellAllHandler] 获取报价失败, in: %s, out: %s, amount: %s, %v",
			token, h.svcCtx.Config.Chain.StablecoinSymbol, uiAmount, err)
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 清仓失败, 请手动清仓", 1)
		return
	}

	// 发送交易
	uiOutAmount := evm.ParseUnits(tx.OutAmount(), h.svcCtx.Config.Chain.StablecoinDecimals)
	quotePrice := uiOutAmount.Div(uiAmount)
	hash, nonce, err := tx.Swap(ctx)
	if err != nil {
		logger.Errorf("[SellAllHandler] 清仓代币 - 发送交易失败, user: %d, inToken: %s, inputAmount: %s, outAmount: %s, hash: %s, %v",
			userId, token, uiAmount, uiOutAmount, hash, err)
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 清仓失败, 请手动清仓", 1)
		return
	}

	logger.Infof("[SellAllHandler] 清仓代币 - 提交交易成功, user: %d, token: %s, totalAmount: %s, hash: %s",
		userId, uiAmount, uiOutAmount, hash)

	// 保存订单记录
	orderArgs := ent.Order{
		Account:    tx.Signer(),
		Token:      token,
		Symbol:     symbol,
		StrategyId: "",
		Type:       order.TypeSell,
		Price:      quotePrice,
		FinalPrice: quotePrice,
		InAmount:   uiAmount,
		OutAmount:  uiOutAmount,
		Status:     order.StatusPending,
		Nonce:      nonce,
		TxHash:     hash,
	}

	_, err = h.svcCtx.OrderModel.Save(ctx, orderArgs)
	if err != nil {
		logger.Errorf("[SellAllHandler] 清仓网格 - 保存订单失败, order: %+v, %v", orderArgs, err)
	}
}
