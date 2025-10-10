package positionhandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/fachebot/evm-grid-bot/internal/dexagg/okxweb3"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot/handler/wallethandler"
	"github.com/fachebot/evm-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/evm-grid-bot/internal/utils"
	"github.com/fachebot/evm-grid-bot/internal/utils/format"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

func InitRoutes(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI, router *pathrouter.Router) {
	NewPositionHomeHandler(svcCtx, botApi).AddRouter(router)
	NewSellAllHandler(svcCtx, botApi).AddRouter(router)
}

type PositionHomeHandler struct {
	botApi *tgbotapi.BotAPI
	svcCtx *svc.ServiceContext
}

func NewPositionHomeHandler(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI) *PositionHomeHandler {
	return &PositionHomeHandler{botApi: botApi, svcCtx: svcCtx}
}

func (h PositionHomeHandler) FormatPath(page int) string {
	return fmt.Sprintf("/position/%d", page)
}

func (h *PositionHomeHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/position", h.handle)
	router.HandleFunc("/position/{page:[0-9]+}", h.handle)
}

func (h *PositionHomeHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tgbotapi.Update) error {
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

	if page == 0 {
		return nil
	}

	w, err := wallethandler.GetUserWallet(ctx, h.svcCtx, userId)
	if err != nil {
		return err
	}

	// 获取代币余额
	okxClient := okxweb3.NewClient(
		h.svcCtx.Config.OkxWeb3.Apikey,
		h.svcCtx.Config.OkxWeb3.Secretkey,
		h.svcCtx.Config.OkxWeb3.Passphrase,
		h.svcCtx.TransportProxy,
	)
	chainIndex := strconv.Itoa(int(h.svcCtx.Config.Chain.Id))
	tokenBalances, err := okxClient.GetAllTokenBalancesByAddress(ctx, chainIndex, w.Account)
	if err != nil {
		return err
	}
	tokenBalances = lo.Filter(tokenBalances, func(tb okxweb3.TokenBalance, _ int) bool {
		return tb.TokenContractAddress != "" &&
			!strings.EqualFold(tb.TokenContractAddress, h.svcCtx.Config.Chain.StablecoinCA) &&
			tb.TokenPrice.GreaterThan(decimal.Zero) &&
			tb.Balance.GreaterThan(decimal.Zero)
	})

	// 计算总页数
	const limit = 10
	totalPage := len(tokenBalances) / limit
	if len(tokenBalances)%limit != 0 {
		totalPage += 1
	}
	if page > totalPage {
		page = totalPage
	}

	// 生成代币列表
	var labels []string
	chainId := h.svcCtx.Config.Chain.Id
	for idx, tokenBalance := range lo.Slice(tokenBalances, (page-1)*limit, (page-1)*limit+limit) {
		number := (page-1)*limit + idx + 1
		labels = append(labels, fmt.Sprintf("%d. [%s](%s) - 余额: `%s`, 价格: `%s`, 价值: `%s`U `%s`",
			number, tokenBalance.Symbol, utils.GetGmgnTokenLink(chainId, tokenBalance.TokenContractAddress), tokenBalance.Balance.Truncate(4),
			format.Price(tokenBalance.TokenPrice, 5), format.Price(tokenBalance.TokenPrice.Mul(tokenBalance.Balance), 5), tokenBalance.TokenContractAddress))
	}

	// 多页翻页功能
	var pageButtons []tgbotapi.InlineKeyboardButton
	if len(tokenBalances) > limit {
		nextPage := page + 1
		previousPage := page - 1
		if previousPage < 1 {
			page = 1
			previousPage = 0
		}
		if nextPage > totalPage {
			page = totalPage
			nextPage = 0
		}
		pageButtons = []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("⬅️ 上一页", h.FormatPath(previousPage)),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d/%d", page, totalPage), h.FormatPath(0)),
			tgbotapi.NewInlineKeyboardButtonData("➡️ 下一页", h.FormatPath(nextPage)),
		}
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	if len(pageButtons) > 0 {
		rows = append(rows, pageButtons)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 刷新界面", h.FormatPath(1)),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("♻️ 代币清仓", SellAllHandler{}.FormatPath()),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ 返回主页", "/home"),
	))

	text := fmt.Sprintf("%s 网格机器人 | 仓位列表\n\n%s", utils.GetNetworkName(chainId), strings.Join(labels, "\n\n"))
	text = text + "\n\n⚠️ 清仓操作不可撤销，谨慎操作！\n⚠️ 重复清仓失败，与代币流动性有关"
	markup := tgbotapi.NewInlineKeyboardMarkup(rows...)
	_, err = utils.ReplyMessage(h.botApi, update, text, markup)
	if err != nil {
		logger.Debugf("[PositionHomeHandler] 处理主页失败, %v", err)
	}

	return nil
}
