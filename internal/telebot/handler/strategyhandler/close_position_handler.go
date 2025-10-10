package strategyhandler

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/evm-grid-bot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ClosePositionyHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewClosePositionyHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *ClosePositionyHandler {
	return &ClosePositionyHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h ClosePositionyHandler) FormatPath(guid string) string {
	return fmt.Sprintf("/strategy/sellall/%s", guid)
}

func (h *ClosePositionyHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/sellall/{uuid}", h.handle)
	router.HandleFunc("/strategy/sellall/{uuid}/{confirm}", h.handle)
}

func (h *ClosePositionyHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	record, err := h.svcCtx.StrategyModel.FindByUserIdGUID(ctx, userId, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyHomeMenu(ctx, h.svcCtx, h.botApi, userId, update, 1)
		}
		logger.Errorf("[ClosePositionyHandler] 查询策略失败, id: %s, %v", guid, err)
		return nil
	}

	_, confirm := vars["confirm"]
	if !confirm {
		text := GetStrategyDetailsText(ctx, h.svcCtx, record)
		rows := [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔴 确认清仓", h.FormatPath(guid)+"/ok"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("◀️ 返回上级", StrategyDetailsHandler{}.FormatPath(record.GUID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🟣 我点错了", StrategyDetailsHandler{}.FormatPath(record.GUID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🟢 取消清仓", StrategyDetailsHandler{}.FormatPath(record.GUID)),
			),
		}
		rand.Shuffle(len(rows), func(i, j int) {
			rows[i], rows[j] = rows[j], rows[i]
		})
		markup := tgbotapi.NewInlineKeyboardMarkup(rows...)
		_, err = utils.ReplyMessage(h.botApi, update, text, markup)
		return err
	} else {
		chatId, _ := utils.GetChatId(&update)
		data, err := h.svcCtx.GridModel.FindByStrategyId(ctx, record.GUID)
		if err != nil {
			logger.Errorf("[ClosePositionyHandler] 获取网格列表失败, strategy: %s, %v", record.GUID, err)
			return err
		}

		ClosePosition(ctx, h.svcCtx, h.botApi, userId, chatId, record, data)

		return DisplayStrategyDetailsMenu(ctx, h.svcCtx, h.botApi, userId, update, record)
	}
}
