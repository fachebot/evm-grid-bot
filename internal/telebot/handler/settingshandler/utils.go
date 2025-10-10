package settingshandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/settings"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getSettingsMenuText(chainId int64) string {
	items := []string{
		"1️⃣ *聚合器:* 指定使用的去中心化交易所聚合器",
		"2️⃣ *交易滑点:* 交易允许的价格滑点",
	}

	text := fmt.Sprintf("%s 网格机器人 | 用户配置", utils.GetNetworkName(chainId))
	text = fmt.Sprintf("%s\n\n%s", text, strings.Join(items, "\n"))
	return text
}

func getUserSettings(ctx context.Context, svcCtx *svc.ServiceContext, userId int64) (*ent.Settings, error) {
	record, err := svcCtx.SettingsModel.FindByUserId(ctx, userId)
	if err == nil {
		return record, nil
	}

	if !ent.IsNotFound(err) {
		return nil, err
	}

	c := svcCtx.Config.Chain
	enableInfiniteApproval := true
	args := ent.Settings{
		UserId:                 userId,
		SlippageBps:            c.SlippageBps,
		DexAggregator:          settings.DexAggregator(c.DexAggregator),
		EnableInfiniteApproval: &enableInfiniteApproval,
	}
	return svcCtx.SettingsModel.Save(ctx, args)
}

func displaySettingsMenu(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI, update tgbotapi.Update, record *ent.Settings) error {
	text := getSettingsMenuText(svcCtx.Config.Chain.Id)
	sellSlippageBps := float64(record.SlippageBps) / 10000 * 100
	if record.SellSlippageBps != nil {
		sellSlippageBps = float64(*record.SellSlippageBps) / 10000 * 100
	}

	exitSlippageBps := sellSlippageBps
	if record.ExitSlippageBps != nil {
		exitSlippageBps = float64(*record.ExitSlippageBps) / 10000 * 100
	}

	enableInfiniteApproval := "🔴 关闭代币无限授权"
	if record.EnableInfiniteApproval != nil && *record.EnableInfiniteApproval {
		enableInfiniteApproval = "🟢 打开代币无限授权"
	}

	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("聚合器: %s", record.DexAggregator), SetDexAggHandler{}.FormatPath()),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				enableInfiniteApproval, SettingsHomeHandler{}.FormatPath(&SettingsOptionEnableInfiniteApproval)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("买入滑点: %v%%", float64(record.SlippageBps)/10000*100), SettingsHomeHandler{}.FormatPath(&SettingsOptionSlippageBps)),
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("卖出滑点: %v%%", sellSlippageBps), SettingsHomeHandler{}.FormatPath(&SettingsOptionSellSlippageBps)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("清仓交易滑点: %v%%", exitSlippageBps), SettingsHomeHandler{}.FormatPath(&SettingsOptionExitSlippageBps)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ 返回主页", "/home"),
		),
	)
	_, err := utils.ReplyMessage(botApi, update, text, markup)
	return err
}
