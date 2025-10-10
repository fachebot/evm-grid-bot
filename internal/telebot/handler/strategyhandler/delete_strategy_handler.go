package strategyhandler

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/strategy"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/evm-grid-bot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type DeleteStrategyHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewDeleteStrategyHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *DeleteStrategyHandler {
	return &DeleteStrategyHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h DeleteStrategyHandler) FormatPath(guid string) string {
	return fmt.Sprintf("/strategy/delete/%s", guid)
}

func (h *DeleteStrategyHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/delete/{uuid}", h.handle)
	router.HandleFunc("/strategy/delete/{uuid}/{confirm}", h.handle)
}

func (h *DeleteStrategyHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	record, err := h.svcCtx.StrategyModel.FindByUserIdGUID(ctx, userId, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyHomeMenu(ctx, h.svcCtx, h.botApi, userId, update, 1)
		}
		logger.Errorf("[DeleteStrategyHandler] 查询策略失败, id: %s, %v", guid, err)
		return nil
	}

	chatId, _ := utils.GetChatId(&update)
	if record.Status == strategy.StatusActive {
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 删除网格之前, 请先关闭策略开关", 1)
		return nil
	}

	_, confirm := vars["confirm"]
	if !confirm {
		text := GetStrategyDetailsText(ctx, h.svcCtx, record)
		rows := [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔴 删除策略", h.FormatPath(guid)+"/confirm"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("◀️ 返回上级", StrategyDetailsHandler{}.FormatPath(record.GUID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🟣 我点错了", StrategyDetailsHandler{}.FormatPath(record.GUID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🟢 取消删除", StrategyDetailsHandler{}.FormatPath(record.GUID)),
			),
		}
		rand.Shuffle(len(rows), func(i, j int) {
			rows[i], rows[j] = rows[j], rows[i]
		})
		markup := tgbotapi.NewInlineKeyboardMarkup(rows...)
		_, err = utils.ReplyMessage(h.botApi, update, text, markup)
		return err
	} else {
		text := fmt.Sprintf("✅ *%s* 策略删除成功", strings.TrimRight(record.Symbol, "\u0000"))
		err = h.svcCtx.StrategyModel.Delete(ctx, record.ID)
		if err != nil {
			text = fmt.Sprintf("❌ *%s* 策略删除失败, 请稍后再试", strings.TrimRight(record.Symbol, "\u0000"))
			logger.Errorf("[DeleteStrategyHandler] 删除策略失败, id: %d, token: %s, %v", record.ID, record.Token, err)
		} else {
			err = DisplayStrategyHomeMenu(ctx, h.svcCtx, h.botApi, userId, update, 1)
			if err != nil {
				logger.Warnf("[DeleteStrategyHandler] 处理主页失败, %v", err)
			}
		}

		utils.SendMessageAndDelayDeletion(h.botApi, chatId, text, 1)

		return nil
	}
}
