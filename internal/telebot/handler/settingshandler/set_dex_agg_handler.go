package settingshandler

import (
	"context"
	"fmt"

	"github.com/fachebot/evm-grid-bot/internal/ent/settings"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/evm-grid-bot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type SetDexAggHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewSetDexAggHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *SetDexAggHandler {
	return &SetDexAggHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h SetDexAggHandler) FormatPath(aggregator ...settings.DexAggregator) string {
	if len(aggregator) == 0 {
		return "/settings/aggregator"
	}
	return fmt.Sprintf("/settings/aggregator/%s", aggregator[0].String())
}

func (h *SetDexAggHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/settings/aggregator", h.handle)
	router.HandleFunc("/settings/aggregator/{value}", h.handle)
}

func (h *SetDexAggHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
	// 处理选项列表
	value, ok := vars["value"]
	if !ok {
		text := getSettingsMenuText(h.svcCtx.Config.Chain.Id)
		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("relay", h.FormatPath(settings.DexAggregatorRelay)),
			),
		)
		_, err := utils.ReplyMessage(h.botApi, update, text, markup)
		return err
	}

	// 获取用户设置
	record, err := getUserSettings(ctx, h.svcCtx, userId)
	if err != nil {
		logger.Errorf("[SetDexAggHandler] 查询用户设置失败, userId: %d, %v", userId, err)
		return err
	}

	// 更新优先级别
	dexAggregator := settings.DexAggregator(value)
	if settings.DexAggregatorValidator(dexAggregator) == nil {
		err = h.svcCtx.SettingsModel.UpdateDexAggregator(ctx, record.ID, dexAggregator)
		if err != nil {
			logger.Errorf("[SetDexAggHandler] 更新 PriorityLevel 配置失败, userId: %d, %v", userId, err)
			return err
		}

		record.DexAggregator = dexAggregator
	}

	displaySettingsMenu(h.svcCtx, h.botApi, update, record)
	return nil
}
