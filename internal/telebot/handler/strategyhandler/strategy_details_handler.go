package strategyhandler

import (
	"context"
	"fmt"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type StrategyDetailsHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewStrategyDetailsHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *StrategyDetailsHandler {
	return &StrategyDetailsHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h StrategyDetailsHandler) FormatPath(guid string) string {
	return fmt.Sprintf("/strategy/details/%s", guid)
}

func (h *StrategyDetailsHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/details/{uuid}", h.handle)
}

func (h *StrategyDetailsHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	record, err := h.svcCtx.StrategyModel.FindByUserIdGUID(ctx, userId, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyHomeMenu(ctx, h.svcCtx, h.botApi, userId, update, 1)
		}
		logger.Errorf("[StrategyDetailsHandler] 查询策略失败, id: %s, %v", guid, err)
		return nil
	}

	return DisplayStrategyDetailsMenu(ctx, h.svcCtx, h.botApi, userId, update, record)
}
