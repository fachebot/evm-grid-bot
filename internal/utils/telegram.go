package utils

import (
	"errors"
	"time"

	"github.com/fachebot/evm-grid-bot/internal/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GetChatId(update *tgbotapi.Update) (int64, bool) {
	if update.Message != nil {
		return update.Message.Chat.ID, true
	}

	if update.EditedMessage != nil {
		return update.EditedMessage.Chat.ID, true
	}

	if update.ChannelPost != nil {
		return update.ChannelPost.Chat.ID, true
	}

	if update.EditedChannelPost != nil {
		return update.EditedChannelPost.Chat.ID, true
	}

	if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Chat.ID, true
	}

	return 0, false
}

func ReplyMessage(
	botApi *tgbotapi.BotAPI,
	update tgbotapi.Update,
	text string,
	markup tgbotapi.InlineKeyboardMarkup,
) (tgbotapi.Message, error) {
	var message *tgbotapi.Message
	if update.Message != nil {
		message = update.Message
	} else if update.CallbackQuery != nil {
		message = update.CallbackQuery.Message
	} else {
		return tgbotapi.Message{}, errors.New("update type unsupported")
	}

	var c tgbotapi.Chattable
	if message.From.UserName != botApi.Self.UserName {
		msg := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FileURL("https://gridbot.4everland.store/grid.png"))
		msg.Caption = text
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = markup
		c = &msg
	} else {
		msg := tgbotapi.NewEditMessageCaption(message.Chat.ID, message.MessageID, text)
		msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: markup.InlineKeyboard}
		msg.ParseMode = tgbotapi.ModeMarkdown
		c = &msg
	}

	return botApi.Send(c)
}

func DeleteMessages(botApi *tgbotapi.BotAPI, chatId int64, messagesIds []int, delaySeconds int) {
	delFunc := func() {
		for _, messageId := range messagesIds {
			c := tgbotapi.NewDeleteMessage(chatId, messageId)
			if _, err := botApi.Request(c); err != nil {
				logger.Debugf("[Telegram] 删除消息失败, %v", err)
			}
		}
	}

	if delaySeconds <= 0 {
		delFunc()
	} else {
		time.AfterFunc(time.Second*time.Duration(delaySeconds), delFunc)
	}
}

func SendMessage(botApi *tgbotapi.BotAPI, chatId int64, text string) (tgbotapi.Message, error) {
	c := tgbotapi.NewMessage(chatId, text)
	c.ParseMode = tgbotapi.ModeMarkdown
	c.DisableWebPagePreview = true
	return botApi.Send(c)
}

func SendMessageAndDelayDeletion(botApi *tgbotapi.BotAPI, chatId int64, text string, delaySeconds int) {
	c := tgbotapi.NewMessage(chatId, text)
	c.ParseMode = tgbotapi.ModeMarkdown
	c.DisableWebPagePreview = true
	msg, err := botApi.Send(c)
	if err == nil {
		DeleteMessages(botApi, chatId, []int{msg.MessageID}, delaySeconds)
	} else {
		logger.Debugf("[Telegram] 发送消息失败, chatId: %d, text: %s, %v", chatId, text, err)
	}
}
