package wallethandler

import (
	"context"
	"fmt"

	"github.com/fachebot/evm-grid-bot/internal/cache"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/evm-grid-bot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type KeyExportHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewKeyExportHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *KeyExportHandler {
	return &KeyExportHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h KeyExportHandler) FormatPath(account string) string {
	return fmt.Sprintf("/wallet/export/%s", account)
}

func (h *KeyExportHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/wallet/export/{account}", h.Handle)
}

func (h *KeyExportHandler) Handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
	account, ok := vars["account"]
	if !ok {
		return nil
	}

	w, err := h.svcCtx.WalletModel.FindByAccount(ctx, account)
	if err != nil {
		logger.Errorf("[KeyExportHandler] 根据账户查找钱包识别, account: %s, %v", account, err)
		return nil
	}
	if w.UserId != userId {
		return nil
	}

	if update.CallbackQuery != nil {
		chatId := update.CallbackQuery.Message.Chat.ID

		// 要求设置密码
		if len(w.Password) == 0 {
			text := "🔐 您未设置过密码\n\n💡 请输入8~16位密码, 用于后续敏感操作, 如导出私钥、提现等"
			c := tgbotapi.NewMessage(chatId, text)
			c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

			msg, err := h.botApi.Send(c)
			if err != nil {
				logger.Debugf("[KeyExportHandler] 发送消息失败, %v", err)
			}

			route := cache.RouteInfo{Path: h.FormatPath(account), Context: update.CallbackQuery.Message}
			h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

			return nil
		}

		// 要求输入密码
		text := "🔑 请输入密码...\n\n如忘记密码, 请联系客服重置!"
		c := tgbotapi.NewMessage(chatId, text)
		c.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}

		msg, err := h.botApi.Send(c)
		if err != nil {
			logger.Debugf("[KeyExportHandler] 发送消息失败, %v", err)
		}

		route := cache.RouteInfo{Path: h.FormatPath(account), Context: update.CallbackQuery.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.MessageID, route)

		return nil
	}

	if update.Message != nil {
		chatId := update.Message.Chat.ID

		deleteMessages := []int{update.Message.MessageID}
		if update.Message.ReplyToMessage != nil {
			deleteMessages = append(deleteMessages, update.Message.ReplyToMessage.MessageID)
		}
		utils.DeleteMessages(h.botApi, chatId, deleteMessages, 0)

		// 设置用户密码
		if len(w.Password) == 0 {
			password := update.Message.Text
			if len(password) < 8 || len(password) > 16 {
				utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 密码长度在8~16位之间", 1)
				return nil
			}

			err = h.svcCtx.WalletModel.UpdatePassword(ctx, account, password)
			if err != nil {
				logger.Errorf("[KeyExportHandler] 更新密码失败, account: %s, password: %s, %v", account, password, err)
			}

			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "🎯 密码设置成功!", 1)

			return nil
		}

		// 验证用户密码
		password := update.Message.Text
		if password != w.Password {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 密码错误, 请检查后再试", 1)
			return nil
		}

		// 解密真正私钥
		pk, err := h.svcCtx.HashEncoder.Decryption(w.PrivateKey)
		if err != nil {
			utils.SendMessageAndDelayDeletion(h.botApi, chatId, "❌ 解密私钥失败, 请联系客服", 1)
			return nil
		}

		mid := len(pk) / 2
		part1 := pk[:mid]
		part2 := pk[mid:]
		text := "🔐 密码验证成功, 请保存您的私钥\n\n⚠️ 安全提示: \n您的私钥已拆分为两部分, 防止恶意软件窃取剪切板数据\n\n🔑 第一部分私钥: \n`%s`\n\n🔑 第二部分私钥: \n`%s`\n\n💾 请立即妥善保存到安全位置\n⏳ 本条消息将在30秒后自动删除"
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, fmt.Sprintf(text, part1, part2), 30)
	}

	return nil
}
