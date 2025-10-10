package strategyhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/fachebot/evm-grid-bot/internal/cache"
	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/strategy"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/evm-grid-bot/internal/utils"
	"github.com/fachebot/evm-grid-bot/internal/utils/evm"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type NewStrategyHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewNewStrategyHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *NewStrategyHandler {
	return &NewStrategyHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h NewStrategyHandler) FormatPath() string {
	return "/strategy/new"
}

func (h *NewStrategyHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/new", h.handle)
}

func (h *NewStrategyHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
	guid, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	// 要求输入合约地址
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		c := tgbotapi.NewMessage(chatId, "🔍 网格策略初始化中...\n\n请输入CA地址, 马上开启智能交易!")
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[NewStrategyHandler] 发送消息失败, %v", err)
		}

		route := cache.RouteInfo{Path: h.FormatPath(), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 检查合约输入地址
	if update.Message != nil {
		chatId := update.Message.Chat.ID

		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		tokenAddress := update.Message.Text

		// 是否重复创建
		record, err := h.svcCtx.StrategyModel.FindByUserIdToken(ctx, userId, tokenAddress)
		if !ent.IsNotFound(err) {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, fmt.Sprintf("❌ %s 策略已存在", tokenAddress), 3)
			return DisplayStrategyDetailsMenu(ctx, h.svcCtx, h.botApi, userId, update, record)
		}

		// 查询合约信息
		tokenMeta, err := evm.GetTokenMeta(ctx, h.svcCtx.EthClient, tokenAddress)
		if err != nil {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, fmt.Sprintf("❌ %s CA地址无效", tokenAddress), 3)
			return nil
		}

		utils.SendMessageAndDelayDeletion(h.botApi, chatId, fmt.Sprintf("♻️ %s 正在初始化网格策略...", tokenAddress), 3)

		// // 查询代币信息
		// jupClient := jupiter.NewJupiterClient(jupConf.Url, jupConf.Apikey, h.svcCtx.TransportProxy)
		// tokenStats, err := jupClient.TokenStats(ctx, tokenAddress)
		// if err != nil {
		// 	logger.Errorf("[NewStrategyHandler] 查询代币信息失败, token: %s, %v", tokenAddress, err)
		// 	utils.SendMessageAndDelayDeletion(h.botApi, chatId, fmt.Sprintf("❌ %s 内部错误，请稍后再试", tokenAddress), 3)
		// 	return nil
		// }

		// // 查询地址聚簇
		// clusterAmountPercentage := decimal.Zero
		// requirements := h.svcCtx.Config.TokenRequirements
		// if requirements.MaxClusterAmountPercentage.GreaterThan(decimal.Zero) {
		// 	faster100xClient := faster100x.NewClient(h.svcCtx.TransportProxy)
		// 	graph, err := faster100xClient.GetLatestGraph(ctx, faster100x.SolanaChain, tokenAddress)
		// 	if err != nil {
		// 		logger.Errorf("[NewStrategyHandler] 查询代币查询地址聚簇, token: %s, %v", tokenAddress, err)
		// 		utils.SendMessageAndDelayDeletion(h.botApi, chatId, fmt.Sprintf("❌ %s 内部错误，请稍后再试", tokenAddress), 3)
		// 		return nil
		// 	}
		// 	clusterAmountPercentage = faster100x.GetClusterAmountPercentage(graph).Mul(decimal.NewFromInt(100))
		// }

		// // 验证代币要求
		// if requirements.MinHolderCount > 0 && tokenStats.HolderCount < requirements.MinHolderCount {
		// 	utils.SendMessageAndDelayDeletion(
		// 		h.botApi, chatId, fmt.Sprintf("❌ %s 持有人数 %d，最低持有人数 %d", tokenAddress, tokenStats.HolderCount, requirements.MinHolderCount), 3)
		// 	return nil
		// }
		// if requirements.MinMarketCap.GreaterThan(decimal.Zero) && tokenStats.MCap.LessThan(requirements.MinMarketCap) {
		// 	text := fmt.Sprintf("❌ %s 不满足必要条件，市值 %v，最低市值 %v", tokenAddress, humanize.Comma(tokenStats.MCap.IntPart()), humanize.Comma(requirements.MinMarketCap.IntPart()))
		// 	utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 3)
		// 	return nil
		// }
		// tokenAgeMinutes := int(time.Since(tokenStats.FirstPool.CreatedAt) / time.Minute)
		// if requirements.MinTokenAgeMinutes > 0 && tokenAgeMinutes < requirements.MinTokenAgeMinutes {
		// 	utils.SendMessageAndDelayDeletion(
		// 		h.botApi, chatId, fmt.Sprintf("❌ %s 不满足必要条件，年龄 %d 分钟，最低年龄 %d", tokenAddress, tokenAgeMinutes, requirements.MinTokenAgeMinutes), 3)
		// 	return nil
		// }
		// if requirements.MaxTokenAgeMinutes > 0 && tokenAgeMinutes > requirements.MaxTokenAgeMinutes {
		// 	utils.SendMessageAndDelayDeletion(
		// 		h.botApi, chatId, fmt.Sprintf("❌ %s 不满足必要条件，年龄 %d 分钟，最高年龄 %d", tokenAddress, tokenAgeMinutes, requirements.MaxTokenAgeMinutes), 3)
		// 	return nil
		// }
		// if requirements.MaxClusterAmountPercentage.GreaterThan(decimal.Zero) && clusterAmountPercentage.GreaterThan(requirements.MaxClusterAmountPercentage) {
		// 	text := fmt.Sprintf("❌ %s 不满足必要条件，聚簇地址持仓占比 %v%%，最高允许占比 %v%%", tokenAddress, clusterAmountPercentage.Truncate(2), requirements.MaxClusterAmountPercentage.Truncate(2))
		// 	utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 3)
		// 	return nil
		// }

		c := h.svcCtx.Config.DefaultGridSettings
		args := ent.Strategy{
			GUID:                   guid.String(),
			UserId:                 userId,
			Token:                  tokenAddress,
			Symbol:                 strings.TrimRight(tokenMeta.Symbol, "\u0000"),
			MartinFactor:           1,
			TakeProfitRatio:        c.TakeProfitRatio,
			UpperPriceBound:        decimal.Zero,
			LowerPriceBound:        decimal.Zero,
			InitialOrderSize:       c.OrderSize,
			LastKlineVolume:        &c.LastKlineVolume,
			FiveKlineVolume:        &c.FiveKlineVolume,
			MaxGridLimit:           &c.MaxGridLimit,
			StopLossExit:           &c.StopLossExit,
			TakeProfitExit:         &c.TakeProfitExit,
			GlobalTakeProfitRatio:  &c.GlobalTakeProfitRatio,
			DropOn:                 c.DropOn,
			CandlesToCheck:         c.CandlesToCheck,
			DropThreshold:          &c.DropThreshold,
			EnableAutoBuy:          true,
			EnableAutoSell:         true,
			EnableAutoExit:         c.EnableAutoExit,
			EnablePushNotification: true,
			Status:                 strategy.StatusInactive,
		}
		record, err = h.svcCtx.StrategyModel.Save(ctx, args)
		if err != nil {
			logger.Errorf("[NewStrategyHandler] 保存策略失败, %v", err)
			return err
		}

		utils.SendMessageAndDelayDeletion(h.botApi, chatId, fmt.Sprintf("✅ %s 网格策略初始化完成", tokenAddress), 3)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategyDetailsMenu(ctx, h.svcCtx, h.botApi, userId, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				update = tgbotapi.Update{Message: route.Context}
				return DisplayStrategyDetailsMenu(ctx, h.svcCtx, h.botApi, userId, update, record)
			}
			return DisplayStrategyDetailsMenu(ctx, h.svcCtx, h.botApi, userId, update, record)
		}
	}

	return nil
}
