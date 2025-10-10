package strategyhandler

import (
	"context"
	"fmt"

	"github.com/fachebot/evm-grid-bot/internal/engine"
	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/strategy"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/model"
	gridstrategy "github.com/fachebot/evm-grid-bot/internal/strategy"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/evm-grid-bot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shopspring/decimal"
)

type StopType string

var (
	StopTypeStop  StopType = "stop"
	StopTypeClear StopType = "clear"
)

type StrategySwitchHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewStrategySwitchHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *StrategySwitchHandler {
	return &StrategySwitchHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h StrategySwitchHandler) FormatPath(guid string) string {
	return fmt.Sprintf("/strategy/switch/%s", guid)
}

func (h StrategySwitchHandler) FormatStopPath(guid string, stopType StopType) string {
	return fmt.Sprintf("/strategy/switch/%s/%s", guid, stopType)
}

func (h *StrategySwitchHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/switch/{uuid}", h.handle)
	router.HandleFunc("/strategy/switch/{uuid}/{stop}", h.handle)
}

func (h *StrategySwitchHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	record, err := h.svcCtx.StrategyModel.FindByUserIdGUID(ctx, userId, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyHomeMenu(ctx, h.svcCtx, h.botApi, userId, update, 1)
		}
		logger.Errorf("[StrategySwitchHandler] 查询策略失败, id: %s, %v", guid, err)
		return nil
	}

	if record.UserId != userId {
		return nil
	}

	// 策略开关
	stopType, ok := vars["stop"]
	if !ok {
		// 关闭策略菜单
		if record.Status == strategy.StatusActive {
			text := GetStrategyDetailsText(ctx, h.svcCtx, record)
			markup := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("❌ 取消关闭", StrategyDetailsHandler{}.FormatPath(guid)),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("1️⃣ 仅关闭策略", h.FormatStopPath(guid, StopTypeStop)),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("2️⃣ 关闭并清仓", h.FormatStopPath(guid, StopTypeClear)),
				),
			)
			_, err = utils.ReplyMessage(h.botApi, update, text, markup)
			return err
		}

		// 处理开启策略
		if record.Status == strategy.StatusInactive {
			return h.handleStartStrategy(ctx, userId, update, record)
		}
		return nil
	}

	// 策略关闭
	switch StopType(stopType) {
	case StopTypeStop:
		return h.handleStopStrategy(ctx, userId, update, record)
	case StopTypeClear:
		return h.handleStopStrategyAndExit(ctx, userId, update, record)
	}

	return nil
}

func (h *StrategySwitchHandler) handleStartStrategy(ctx context.Context, userId int64, update tgbotapi.Update, record *ent.Strategy) error {
	chatId, ok := utils.GetChatId(&update)
	if !ok {
		return nil
	}

	if record.InitialOrderSize.LessThanOrEqual(decimal.Zero) {
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 开启策略失败, 单笔投资金额必须大于0", 1)
		return nil
	}

	if record.LowerPriceBound.LessThanOrEqual(decimal.Zero) {
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 开启策略失败, 网格价格下限必须大于0", 1)
		return nil
	}

	if record.UpperPriceBound.LessThanOrEqual(decimal.Zero) {
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 开启策略失败, 网格价格上限必须大于0", 1)
		return nil
	}

	if record.UpperPriceBound.LessThanOrEqual(record.LowerPriceBound) {
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 开启策略失败, 网格价格上限必须大于价格下限", 1)
		return nil
	}

	utils.SendMessageAndDelayDeletion(h.botApi, chatId, "✅ 正在开启策略, 请稍后...", 1)

	err := utils.Tx(ctx, h.svcCtx.DbClient, func(tx *ent.Tx) error {
		_, err := model.NewGridModel(tx.Grid).DeleteByStrategyId(ctx, record.GUID)
		if err != nil {
			return err
		}

		err = model.NewStrategyModel(tx.Strategy).UpdateGridTrend(ctx, record.ID, "")
		if err != nil {
			return err
		}

		err = model.NewStrategyModel(tx.Strategy).ClearLastLowerThresholdAlertTime(ctx, record.ID)
		if err != nil {
			return err
		}

		err = model.NewStrategyModel(tx.Strategy).ClearLastUpperThresholdAlertTime(ctx, record.ID)
		if err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatusByGuid(ctx, record.GUID, strategy.StatusActive)
	})
	if err != nil {
		logger.Errorf("[StrategySwitchHandler] 更新策略状态失败, id: %s, %v", record.GUID, err)
		return err
	}

	record.Status = strategy.StatusActive

	s := gridstrategy.NewGridStrategy(h.svcCtx, record)
	err = h.svcCtx.Engine.StartStrategy([]engine.Strategy{s})
	if err != nil {
		logger.Errorf("[StrategySwitchHandler] 开启策略失败, id: %s, %v", record.GUID, err)
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 开启策略失败, 请稍后再试", 1)
		return err
	}

	utils.SendMessageAndDelayDeletion(h.botApi, chatId, "✅ 策略已开启", 1)

	logger.Debugf("[StrategySwitchHandler] 策略已开启, id: %s", record.GUID)

	return DisplayStrategyDetailsMenu(ctx, h.svcCtx, h.botApi, userId, update, record)
}

func (h *StrategySwitchHandler) handleStopStrategy(ctx context.Context, userId int64, update tgbotapi.Update, record *ent.Strategy) error {
	chatId, ok := utils.GetChatId(&update)
	if !ok {
		return nil
	}

	utils.SendMessageAndDelayDeletion(h.botApi, chatId, "✅ 正在关闭策略, 请稍后...", 1)

	err := utils.Tx(ctx, h.svcCtx.DbClient, func(tx *ent.Tx) error {
		_, err := model.NewGridModel(tx.Grid).DeleteByStrategyId(ctx, record.GUID)
		if err != nil {
			return err
		}

		err = model.NewStrategyModel(tx.Strategy).UpdateFirstOrderId(ctx, record.ID, nil)
		if err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatusByGuid(ctx, record.GUID, strategy.StatusInactive)
	})
	if err != nil {
		logger.Errorf("[StrategySwitchHandler] 更新策略状态失败, id: %s, %v", record.GUID, err)
		return err
	}

	record.Status = strategy.StatusInactive

	h.svcCtx.Engine.StopStrategy(record.GUID)
	utils.SendMessageAndDelayDeletion(h.botApi, chatId, "✅ 策略已关闭", 1)

	logger.Debugf("[StrategySwitchHandler] 策略已关闭, id: %s", record.GUID)

	return DisplayStrategyDetailsMenu(ctx, h.svcCtx, h.botApi, userId, update, record)
}

func (h *StrategySwitchHandler) handleStopStrategyAndExit(ctx context.Context, userId int64, update tgbotapi.Update, record *ent.Strategy) error {
	chatId, ok := utils.GetChatId(&update)
	if !ok {
		return nil
	}

	data, err := h.svcCtx.GridModel.FindByStrategyId(ctx, record.GUID)
	if err != nil {
		logger.Errorf("[StrategySwitchHandler] 获取网格列表失败, strategy: %s, %v", record.GUID, err)
		return err
	}

	if err = h.handleStopStrategy(ctx, userId, update, record); err != nil && record.Status != strategy.StatusInactive {
		return err
	}

	ClosePosition(ctx, h.svcCtx, h.botApi, userId, chatId, record, data)

	return nil
}
