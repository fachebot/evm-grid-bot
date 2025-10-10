package strategyhandler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func InitRoutes(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI, router *pathrouter.Router) {
	NewStrategyHomeHandler(svcCtx, botApi).AddRouter(router)
	NewNewStrategyHandler(svcCtx, botApi).AddRouter(router)
	NewStrategyDetailsHandler(svcCtx, botApi).AddRouter(router)
	NewStrategySettingsHandler(svcCtx, botApi).AddRouter(router)
	NewDeleteStrategyHandler(svcCtx, botApi).AddRouter(router)
	NewStrategySwitchHandler(svcCtx, botApi).AddRouter(router)
	NewStrategyTradesHandler(svcCtx, botApi).AddRouter(router)
	NewClosePositionyHandler(svcCtx, botApi).AddRouter(router)
	NewQuickStartStrategyHandler(svcCtx, botApi).AddRouter(router)
}

type StrategyHomeHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewStrategyHomeHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *StrategyHomeHandler {
	return &StrategyHomeHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h StrategyHomeHandler) FormatPath(page int) string {
	return fmt.Sprintf("/strategy/%d", page)
}

func (h *StrategyHomeHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy", h.handle)
	router.HandleFunc("/strategy/{page:[0-9]+}", h.handle)
}

func (h *StrategyHomeHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
	var page int
	val, ok := vars["page"]
	if !ok {
		page = 1
	} else {
		n, err := strconv.Atoi((val))
		if err != nil {
			page = 1
		} else {
			page = n
		}
	}

	err := DisplayStrategyHomeMenu(ctx, h.svcCtx, h.botApi, userId, update, page)
	if err != nil {
		logger.Warnf("[StrategyHomeHandler] 处理主页失败, %v", err)
	}

	return nil
}
