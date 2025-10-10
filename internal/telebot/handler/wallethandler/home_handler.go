package wallethandler

import (
	"context"

	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func InitRoutes(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI, router *pathrouter.Router) {
	NewWalletHomeHandler(svcCtx, botApi).AddRouter(router)
	NewKeyExportHandler(svcCtx, botApi).AddRouter(router)
}

type WalletHomeHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewWalletHomeHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *WalletHomeHandler {
	return &WalletHomeHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h WalletHomeHandler) FormatPath() string {
	return "/wallet"
}

func (h *WalletHomeHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/wallet", h.Handle)
}

func (h *WalletHomeHandler) Handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
	err := DisplayWalletMenu(ctx, h.svcCtx, h.botApi, userId, update)
	if err != nil {
		logger.Debugf("[WalletHomeHandler] 处理主页失败, %v", err)
	}

	return nil
}
