package strategyhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/strategy"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/evm-grid-bot/internal/utils"
	"github.com/fachebot/evm-grid-bot/internal/utils/evm"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
)

type QuickStartStrategyHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewQuickStartStrategyHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *QuickStartStrategyHandler {
	return &QuickStartStrategyHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h QuickStartStrategyHandler) FormatPath(token string) string {
	return fmt.Sprintf("/strategy/quickstart/%s", token)
}

func (h *QuickStartStrategyHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/quickstart/{token}", h.handle)
}

func (h *QuickStartStrategyHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
	if update.Message == nil {
		return nil
	}

	guid, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	// 获取合约地址
	chatId := update.Message.Chat.ID
	tokenAddress, ok := vars["token"]
	if !ok {
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, fmt.Sprintf("❌ %s CA地址无效", tokenAddress), 1)
		return nil
	}

	// 是否重复创建
	record, err := h.svcCtx.StrategyModel.FindByUserIdToken(ctx, userId, tokenAddress)
	if !ent.IsNotFound(err) {
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, fmt.Sprintf("❌ %s 策略已存在", tokenAddress), 3)
		return DisplayStrategyDetailsMenu(ctx, h.svcCtx, h.botApi, userId, update, record)
	}

	// 查询合约信息
	tokenMeta, err := evm.GetTokenMeta(ctx, h.svcCtx.EthClient, tokenAddress)
	if err != nil {
		utils.SendMessageAndDelayDeletion(h.botApi, chatId, fmt.Sprintf("❌ %s CA地址无效", tokenAddress), 1)
		return nil
	}

	// 保存策略信息
	c := h.svcCtx.Config.QuickStartSettings
	args := ent.Strategy{
		GUID:                   guid.String(),
		UserId:                 userId,
		Token:                  tokenAddress,
		Symbol:                 strings.TrimRight(tokenMeta.Symbol, "\u0000"),
		MartinFactor:           1,
		TakeProfitRatio:        c.TakeProfitRatio,
		UpperPriceBound:        c.UpperPriceBound,
		LowerPriceBound:        c.LowerPriceBound,
		InitialOrderSize:       c.OrderSize,
		LastKlineVolume:        &c.LastKlineVolume,
		FiveKlineVolume:        &c.FiveKlineVolume,
		MaxGridLimit:           &c.MaxGridLimit,
		TakeProfitExit:         &c.TakeProfitExit,
		DropOn:                 c.DropOn,
		CandlesToCheck:         c.CandlesToCheck,
		DropThreshold:          &c.DropThreshold,
		EnableAutoBuy:          true,
		EnableAutoSell:         true,
		EnableAutoExit:         c.EnableAutoExit,
		EnablePushNotification: true,
		Status:                 strategy.StatusInactive,
	}
	record, err = h.svcCtx.StrategyModel.Save(ctx, args)
	if err != nil {
		logger.Errorf("[QuickStartStrategyHandler] 保存策略失败, %v", err)
		return err
	}

	utils.SendMessageAndDelayDeletion(h.botApi, chatId, fmt.Sprintf("✅ %s 网格策略初始化完成", tokenAddress), 3)

	// 更新用户界面
	return DisplayStrategSettingsMenu(h.svcCtx, h.botApi, update, record)
}
