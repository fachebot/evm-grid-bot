package strategyhandler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/fachebot/evm-grid-bot/internal/cache"
	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/strategy"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/evm-grid-bot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shopspring/decimal"
)

type SettingsOption int

var (
	SettingsOptionOrderSize              SettingsOption = 1
	SettingsOptionUpperPriceBound        SettingsOption = 2
	SettingsOptionLowerPriceBound        SettingsOption = 3
	SettingsOptionTakeProfitRatio        SettingsOption = 4
	SettingsOptionEnableAutoBuy          SettingsOption = 5
	SettingsOptionEnableAutoSell         SettingsOption = 6
	SettingsOptionEnableAutoClear        SettingsOption = 7
	SettingsOptionEnablePushNotification SettingsOption = 8
	SettingsOptionLastKlineVolume        SettingsOption = 9
	SettingsOptionFiveKlineVolume        SettingsOption = 10
	SettingsOptionMaxGridLimit           SettingsOption = 11
	SettingsOptionTakeProfitExit         SettingsOption = 12
	SettingsOptionDynamicStopLoss        SettingsOption = 13
	SettingsOptionUpperBoundExit         SettingsOption = 14
	SettingsOptionDropOn                 SettingsOption = 15
	SettingsOptionCandlesToCheck         SettingsOption = 16
	SettingsOptionDropThreshold          SettingsOption = 17
	SettingsOptionStopLossExit           SettingsOption = 18
	SettingsOptionGlobalTakeProfitRatio  SettingsOption = 19
)

type StrategySettingsHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewStrategySettingsHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *StrategySettingsHandler {
	return &StrategySettingsHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h StrategySettingsHandler) FormatPath(guid string, option *SettingsOption) string {
	if option == nil {
		return fmt.Sprintf("/strategy/settings/%s", guid)
	}
	return fmt.Sprintf("/strategy/settings/%s/%d", guid, *option)
}

func (h *StrategySettingsHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/settings/{uuid}", h.handle)
	router.HandleFunc("/strategy/settings/{uuid}/{option}", h.handle)
}

func (h *StrategySettingsHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	record, err := h.svcCtx.StrategyModel.FindByUserIdGUID(ctx, userId, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyHomeMenu(ctx, h.svcCtx, h.botApi, userId, update, 1)
		}
		logger.Errorf("[StrategySettingsHandler] 查询策略失败, id: %s, %v", guid, err)
		return nil
	}

	if record.UserId != userId {
		return nil
	}

	option, ok := vars["option"]
	if !ok {
		return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
	}

	optionValue, err := strconv.Atoi(option)
	if err != nil {
		return DisplayStrategyHomeMenu(ctx, h.svcCtx, h.botApi, userId, update, 1)
	}

	switch SettingsOption(optionValue) {
	case SettingsOptionOrderSize:
		return h.handleOrderSize(ctx, update, record)
	case SettingsOptionMaxGridLimit:
		return h.handleMaxGridLimit(ctx, update, record)
	case SettingsOptionUpperPriceBound:
		return h.handleUpperPriceBound(ctx, update, record)
	case SettingsOptionLowerPriceBound:
		return h.handleLowerPriceBound(ctx, update, record)
	case SettingsOptionTakeProfitRatio:
		return h.handleTakeProfitRatio(ctx, update, record)
	case SettingsOptionEnableAutoBuy:
		return h.handleEnableAutoBuy(ctx, update, record)
	case SettingsOptionEnableAutoSell:
		return h.handleEnableAutoSell(ctx, update, record)
	case SettingsOptionEnableAutoClear:
		return h.handleEnableAutoClear(ctx, update, record)
	case SettingsOptionLastKlineVolume:
		return h.handleLastKlineVolume(ctx, update, record)
	case SettingsOptionFiveKlineVolume:
		return h.handleFiveKlineVolume(ctx, update, record)
	case SettingsOptionEnablePushNotification:
		return h.handleEnablePushNotification(ctx, update, record)
	case SettingsOptionUpperBoundExit:
		return h.handleUpperBoundExit(ctx, update, record)
	case SettingsOptionTakeProfitExit:
		return h.handleTakeProfitExit(ctx, update, record)
	case SettingsOptionDynamicStopLoss:
		return h.handleEnableDynamicStopLoss(ctx, update, record)
	case SettingsOptionDropOn:
		return h.handleDropOn(ctx, update, record)
	case SettingsOptionCandlesToCheck:
		return h.handleCandlesToCheck(ctx, update, record)
	case SettingsOptionDropThreshold:
		return h.handleDropThreshold(ctx, update, record)
	case SettingsOptionStopLossExit:
		return h.handleStopLossExit(ctx, update, record)
	case SettingsOptionGlobalTakeProfitRatio:
		return h.handleGlobalTakeProfitRatio(ctx, update, record)
	}

	return nil
}

func (h *StrategySettingsHandler) handleEnableAutoBuy(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	if update.CallbackQuery == nil {
		return nil
	}

	text := "✅ 配置修改成功"
	err := h.svcCtx.StrategyModel.UpdateEnableAutoBuy(ctx, record.ID, !record.EnableAutoBuy)
	if err == nil {
		record.EnableAutoBuy = !record.EnableAutoBuy
	} else {
		text = "❌ 配置修改失败, 请稍后重试"
		logger.Errorf("[StrategySettingsHandler] 更新配置[EnableAutoBuy]失败, %v", err)
	}

	chatId := update.CallbackQuery.Message.Chat.ID
	utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

	return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
}

func (h *StrategySettingsHandler) handleEnableAutoSell(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	if update.CallbackQuery == nil {
		return nil
	}

	text := "✅ 配置修改成功"
	err := h.svcCtx.StrategyModel.UpdateEnableAutoSell(ctx, record.ID, !record.EnableAutoSell)
	if err == nil {
		record.EnableAutoSell = !record.EnableAutoSell
	} else {
		text = "❌ 配置修改失败, 请稍后重试"
		logger.Errorf("[StrategySettingsHandler] 更新配置[EnableAutoSell]失败, %v", err)
	}

	chatId := update.CallbackQuery.Message.Chat.ID
	utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

	return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
}

func (h *StrategySettingsHandler) handleEnableAutoClear(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	if update.CallbackQuery == nil {
		return nil
	}

	text := "✅ 配置修改成功"
	err := h.svcCtx.StrategyModel.UpdateEnableAutoExit(ctx, record.ID, !record.EnableAutoExit)
	if err == nil {
		record.EnableAutoExit = !record.EnableAutoExit
	} else {
		text = "❌ 配置修改失败, 请稍后重试"
		logger.Errorf("[StrategySettingsHandler] 更新配置[EnableAutoExit]失败, %v", err)
	}

	chatId := update.CallbackQuery.Message.Chat.ID
	utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

	return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
}

func (h *StrategySettingsHandler) handleEnablePushNotification(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	if update.CallbackQuery == nil {
		return nil
	}

	text := "✅ 配置修改成功"
	err := h.svcCtx.StrategyModel.UpdateEnablePushNotification(ctx, record.ID, !record.EnablePushNotification)
	if err == nil {
		record.EnablePushNotification = !record.EnablePushNotification
	} else {
		text = "❌ 配置修改失败, 请稍后重试"
		logger.Errorf("[StrategySettingsHandler] 更新配置[EnablePushNotification]失败, %v", err)
	}

	chatId := update.CallbackQuery.Message.Chat.ID
	utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

	return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
}

func (h *StrategySettingsHandler) handleEnableDynamicStopLoss(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	if update.CallbackQuery == nil {
		return nil
	}

	text := "✅ 配置修改成功"
	err := h.svcCtx.StrategyModel.UpdateDynamicStopLoss(ctx, record.ID, !record.DynamicStopLoss)
	if err == nil {
		record.DynamicStopLoss = !record.DynamicStopLoss
	} else {
		text = "❌ 配置修改失败, 请稍后重试"
		logger.Errorf("[StrategySettingsHandler] 更新配置[DynamicStopLoss]失败, %v", err)
	}

	chatId := update.CallbackQuery.Message.Chat.ID
	utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

	return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
}

func (h *StrategySettingsHandler) handleOrderSize(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	// 步骤1
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		text := fmt.Sprintf("🌳 填写单笔买入 %s 金额\n\n💵 例: 200 → 代表每次买入200 %s",
			h.svcCtx.Config.Chain.StablecoinSymbol, h.svcCtx.Config.Chain.StablecoinSymbol)
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionOrderSize), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThanOrEqual(decimal.Zero) {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "⚠️ 请输入有效金额", 1)
			return nil
		} else if d.GreaterThanOrEqual(decimal.NewFromInt(1000)) {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "⚠️ 金额需在0-1000之间", 1)
			return nil
		}

		if d.Equal(record.InitialOrderSize) {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateInitialOrderSize(ctx, record.ID, d)
		if err == nil {
			record.InitialOrderSize = d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[EnablePushNotification]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		}
	}

	return nil
}

func (h *StrategySettingsHandler) handleMaxGridLimit(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	// 步骤1
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		text := "🌳 填写最多持有网格数量, 网格数量达到此值后停止买入"
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionMaxGridLimit), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := strconv.Atoi(update.Message.Text)
		if err != nil || d <= 0 {
			text := "⚠️ 请输入有效的整数"
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)
			return nil
		}

		if (record.MaxGridLimit == nil && d == 0) || (record.MaxGridLimit != nil && d == *record.MaxGridLimit) {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateMaxGridLimit(ctx, record.ID, d)
		if err == nil {
			record.MaxGridLimit = &d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[MaxGridLimit]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		}
	}

	return nil
}

func (h *StrategySettingsHandler) handleTakeProfitRatio(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	chatId, _ := utils.GetChatId(&update)
	if record.Status == strategy.StatusActive {
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 策略开启后, 只允许修改单笔投入金额", 1)
		return nil
	}

	// 步骤1
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		text := "🌳 填写网格止盈间隔%\n\n💵 例如: 10｜代表 10% , 单位是 %"
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionTakeProfitRatio), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThanOrEqual(decimal.Zero) {
			text := "⚠️ 请输入有效止盈间隔"
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)
			return nil
		}

		if d.Equal(record.TakeProfitRatio) {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateTakeProfitRatio(ctx, record.ID, d)
		if err == nil {
			record.TakeProfitRatio = d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[TakeProfitRatio]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		}
	}

	return nil
}

func (h *StrategySettingsHandler) handleUpperPriceBound(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	chatId, _ := utils.GetChatId(&update)
	if record.Status == strategy.StatusActive {
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 策略开启后, 只允许修改单笔投入金额", 1)
		return nil
	}

	// 步骤1
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		text := fmt.Sprintf("🌳 填写网格最高价格（单位: %s）\n\n💵 例: 100 → 代表100 %s",
			h.svcCtx.Config.Chain.StablecoinSymbol, h.svcCtx.Config.Chain.StablecoinSymbol)
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionUpperPriceBound), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThanOrEqual(decimal.Zero) {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "⚠️ 请输入有效金额", 1)
			return nil
		} else if d.LessThanOrEqual(record.LowerPriceBound) {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "⚠️ 网格最高价格必须大于最低价格", 1)
			return nil
		}

		if d.Equal(record.UpperPriceBound) {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateUpperPriceBound(ctx, record.ID, d)
		if err == nil {
			record.UpperPriceBound = d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[UpperPriceBound]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		}
	}

	return nil
}

func (h *StrategySettingsHandler) handleLowerPriceBound(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	chatId, _ := utils.GetChatId(&update)
	if record.Status == strategy.StatusActive {
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 策略开启后, 只允许修改单笔投入金额", 1)
		return nil
	}

	// 步骤1
	if update.CallbackQuery != nil {
		text := fmt.Sprintf("🌳 填写网格最低价格（单位: %s）\n\n💵 例: 100 → 代表100 %s",
			h.svcCtx.Config.Chain.StablecoinSymbol, h.svcCtx.Config.Chain.StablecoinSymbol)
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionLowerPriceBound), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThanOrEqual(decimal.Zero) {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "⚠️ 请输入有效金额", 1)
			return nil
		} else if d.GreaterThanOrEqual(record.UpperPriceBound) {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "⚠️ 网格最低价格必须小于最高价格", 1)
			return nil
		}

		if d.Equal(record.LowerPriceBound) {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateLowerPriceBound(ctx, record.ID, d)
		if err == nil {
			record.LowerPriceBound = d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[LowerPriceBound]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		}
	}

	return nil
}

func (h *StrategySettingsHandler) handleLastKlineVolume(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	// 步骤1
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		text := "🌳 填写近期的最小交易量, 如果交易量小于此值则不会买入, 0 表示不受限制"
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionLastKlineVolume), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThan(decimal.Zero) {
			text := "⚠️ 请输入有效交易量数值"
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)
			return nil
		}

		if (record.LastKlineVolume == nil && d.IsZero()) || (record.LastKlineVolume != nil && d.Equal(*record.LastKlineVolume)) {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateLastKlineVolume(ctx, record.ID, d)
		if err == nil {
			record.LastKlineVolume = &d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[LastKlineVolume]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		}
	}

	return nil
}

func (h *StrategySettingsHandler) handleFiveKlineVolume(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	// 步骤1
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		text := "🌳 填写近期5分钟的最小交易量, 如果交易量小于此值则不会买入, 0 表示不受限制"
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionFiveKlineVolume), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThan(decimal.Zero) {
			text := "⚠️ 请输入有效交易量数值"
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)
			return nil
		}

		if (record.FiveKlineVolume == nil && d.IsZero()) || (record.FiveKlineVolume != nil && d.Equal(*record.FiveKlineVolume)) {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateFiveKlineVolume(ctx, record.ID, d)
		if err == nil {
			record.FiveKlineVolume = &d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[FiveKlineVolume]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		}
	}

	return nil
}

func (h *StrategySettingsHandler) handleUpperBoundExit(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	// 步骤1
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		text := "🌳 填写离场目标价格, 到达此价格将自动清仓并停止策略"
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionUpperBoundExit), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThan(decimal.Zero) {
			text := "⚠️ 请输入有效离场目标价格"
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)
			return nil
		}

		if (record.UpperBoundExit == nil && d.IsZero()) || (record.UpperBoundExit != nil && d.Equal(*record.UpperBoundExit)) {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateUpperBoundExit(ctx, record.ID, d)
		if err == nil {
			record.UpperBoundExit = &d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[UpperBoundExit]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		}
	}

	return nil
}

func (h *StrategySettingsHandler) handleStopLossExit(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	// 步骤1
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		text := "🌳 填写止损金额阈值, 如果亏损金额达到阈值, 将自动清仓并停止策略"
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionStopLossExit), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThan(decimal.Zero) {
			text := "⚠️ 请输入有效止损金额阈值"
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)
			return nil
		}

		if (record.StopLossExit == nil && d.IsZero()) || (record.StopLossExit != nil && d.Equal(*record.StopLossExit)) {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateStopLossExit(ctx, record.ID, d)
		if err == nil {
			record.StopLossExit = &d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[UpdateStopLossExit]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.svcCtx.BotApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.svcCtx.BotApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.svcCtx.BotApi, update, record)
		}
	}

	return nil
}

func (h *StrategySettingsHandler) handleTakeProfitExit(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	// 步骤1
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		text := "🌳 填写盈利目标金额, 如果盈利达到盈利目标, 将自动清仓并停止策略"
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionTakeProfitExit), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThan(decimal.Zero) {
			text := "⚠️ 请输入有效盈利目标金额"
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)
			return nil
		}

		if (record.TakeProfitExit == nil && d.IsZero()) || (record.TakeProfitExit != nil && d.Equal(*record.TakeProfitExit)) {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateTakeProfitExit(ctx, record.ID, d)
		if err == nil {
			record.TakeProfitExit = &d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[TakeProfitExit]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		}
	}

	return nil
}

func (h *StrategySettingsHandler) handleDropOn(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	if update.CallbackQuery == nil {
		return nil
	}

	text := "✅ 配置修改成功"
	err := h.svcCtx.StrategyModel.UpdateDropOn(ctx, record.ID, !record.DropOn)
	if err == nil {
		record.DropOn = !record.DropOn
	} else {
		text = "❌ 配置修改失败, 请稍后重试"
		logger.Errorf("[StrategySettingsHandler] 更新配置[DropOn]失败, %v", err)
	}

	chatId := update.CallbackQuery.Message.Chat.ID
	utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

	return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
}

func (h *StrategySettingsHandler) handleCandlesToCheck(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	// 步骤1
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		text := "🌳 填写防瀑布K线根数, 如果达到下跌阈值则自动关闭策略并清仓"
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionCandlesToCheck), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := strconv.Atoi(update.Message.Text)
		if err != nil || d < 0 {
			text := "⚠️ 请输入有效K线根数"
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)
			return nil
		}

		if d == record.CandlesToCheck {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateCandlesToCheck(ctx, record.ID, d)
		if err == nil {
			record.CandlesToCheck = d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[CandlesToCheck]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		}
	}

	return nil
}

func (h *StrategySettingsHandler) handleDropThreshold(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	// 步骤1
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		text := "🌳 填写防瀑布下跌阈值%, 如果达到下跌阈值则自动关闭策略并清仓\n\n💵 例如: 10｜代表 10% , 单位是 %"
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionDropThreshold), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThan(decimal.Zero) {
			text := "⚠️ 请输入有效防瀑布下跌阈值%"
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)
			return nil
		}

		if (record.DropThreshold == nil && d.IsZero()) || (record.DropThreshold != nil && d.Equal(*record.DropThreshold)) {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateDropThreshold(ctx, record.ID, d)
		if err == nil {
			record.DropThreshold = &d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[DropThreshold]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
		}
	}

	return nil
}

func (h *StrategySettingsHandler) handleGlobalTakeProfitRatio(ctx context.Context, update tgbotapi.Update, record *ent.Strategy) error {
	// 步骤1
	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID
		text := "🌳 填写全局止盈%, 如果整体盈利率达到此值则清仓并关闭策略, 0 表示不受限制\n\n💵 例如: 1｜代表 1% , 单位是 %"
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &SettingsOptionGlobalTakeProfitRatio), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 检查输入金额
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThan(decimal.Zero) {
			text := "⚠️ 请输入有效全局止盈%"
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)
			return nil
		}

		d = d.Div(decimal.NewFromInt(100))
		if (record.GlobalTakeProfitRatio == nil && d.IsZero()) || (record.GlobalTakeProfitRatio != nil && d.Equal(*record.GlobalTakeProfitRatio)) {
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateGlobalTakeProfitRatio(ctx, record.ID, d)
		if err == nil {
			record.GlobalTakeProfitRatio = &d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[GlobalTakeProfitRatio]失败, %v", err)
		}
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		// 更新用户界面
		if update.Message.ReplyToMessage == nil {
			return DisplayStrategSettingsMenu(h.svcCtx, h.svcCtx.BotApi, update, record)
		} else {
			route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyToMessage.MessageID)
			if ok && route.Context != nil {
				return DisplayStrategSettingsMenu(h.svcCtx, h.svcCtx.BotApi, tgbotapi.Update{Message: route.Context}, record)
			}
			return DisplayStrategSettingsMenu(h.svcCtx, h.svcCtx.BotApi, update, record)
		}
	}

	return nil
}
