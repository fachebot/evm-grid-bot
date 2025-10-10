package strategyhandler

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fachebot/evm-grid-bot/internal/charts"
	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/grid"
	"github.com/fachebot/evm-grid-bot/internal/ent/order"
	"github.com/fachebot/evm-grid-bot/internal/ent/strategy"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/model"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/swap"
	"github.com/fachebot/evm-grid-bot/internal/utils"
	"github.com/fachebot/evm-grid-bot/internal/utils/evm"
	"github.com/fachebot/evm-grid-bot/internal/utils/format"

	"github.com/dustin/go-humanize"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

func ClosePosition(ctx context.Context, svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI, userId, chatId int64, record *ent.Strategy, data []*ent.Grid) {
	// 计算总仓位
	uiTotalAmount := decimal.Zero
	uiTotalQuantity := decimal.Zero
	for _, item := range data {
		if item.Status != grid.StatusBought {
			continue
		}
		uiTotalAmount = uiTotalAmount.Add(item.Amount)
		uiTotalQuantity = uiTotalQuantity.Add(item.Quantity)
	}

	w, err := svcCtx.WalletModel.FindByUserId(ctx, userId)
	if err != nil {
		logger.Errorf("[ClosePosition] 查询用户钱包失败, userId: %d, %v", userId, err)
		return
	}

	// 获取元数据
	tokenmeta, err := svcCtx.TokenMetaCache.GetTokenMeta(ctx, record.Token)
	if err != nil {
		logger.Errorf("[ClosePosition] 查询代币元数据失败, userId: %d, %v", userId, err)
		return
	}

	// 获取代币余额
	tokenBalance, err := evm.GetTokenBalance(ctx, svcCtx.EthClient, record.Token, w.Account)
	if err != nil {
		logger.Debugf("[ClosePosition] 获取代币余额失败, token: %s, %v", record.Token, err)
		utils.SendMessageAndDelayDeletion(botApi, chatId, "❌ 清仓失败, 请手动清仓", 1)
		return
	}
	uiTokenBalance := evm.ParseUnits(tokenBalance, tokenmeta.Decimals)
	if uiTotalQuantity.GreaterThan(uiTokenBalance) {
		uiTotalQuantity = uiTokenBalance
	} else if uiTokenBalance.GreaterThan(uiTotalQuantity) {
		uiTotalQuantity = uiTokenBalance
	}

	if uiTotalQuantity.LessThanOrEqual(decimal.Zero) {
		utils.SendMessageAndDelayDeletion(botApi, chatId, "当前代币无持仓 📊", 1)
		return
	}

	utils.SendMessageAndDelayDeletion(botApi, chatId, fmt.Sprintf("📊 代币持仓: %s 枚 | ⚡️ 清仓中...", uiTotalQuantity), 1)

	// 获取报价
	amount := evm.FormatUnits(uiTotalQuantity, tokenmeta.Decimals)
	swapService := swap.NewSwapService(svcCtx, userId)
	tx, err := swapService.Quote(ctx, record.Token, svcCtx.Config.Chain.StablecoinCA, amount, true)
	if err != nil {
		logger.Errorf("[ClosePosition] 获取报价失败, in: %s, out: %s, amount: %s, %v",
			record.Token, svcCtx.Config.Chain.StablecoinSymbol, uiTotalQuantity, err)
		utils.SendMessageAndDelayDeletion(botApi, chatId, "❌ 清仓失败, 请手动清仓", 1)
		return
	}

	// 发送交易
	uiOutAmount := evm.ParseUnits(tx.OutAmount(), svcCtx.Config.Chain.StablecoinDecimals)
	quotePrice := uiOutAmount.Div(uiTotalQuantity)
	hash, nonce, err := tx.Swap(ctx)
	if err != nil {
		logger.Errorf("[ClosePosition] 清仓代币 - 发送交易失败, user: %d, inToken: %s, inputAmount: %s, outAmount: %s, hash: %s, %v",
			userId, record.Token, uiTotalQuantity, uiOutAmount, hash, err)
		utils.SendMessageAndDelayDeletion(botApi, chatId, "❌ 清仓失败, 请手动清仓", 1)
		return
	}

	logger.Infof("[ClosePosition] 清仓代币 - 提交交易成功, user: %d, strategy: %s, totalAmount: %s, hash: %s",
		userId, record.GUID, uiTotalQuantity, hash)

	// 保存订单记录
	orderArgs := ent.Order{
		Account:     tx.Signer(),
		Token:       record.Token,
		Symbol:      record.Symbol,
		StrategyId:  record.GUID,
		Type:        order.TypeSell,
		GridBuyCost: &uiTotalAmount,
		Price:       quotePrice,
		FinalPrice:  quotePrice,
		InAmount:    uiTotalQuantity,
		OutAmount:   uiOutAmount,
		Status:      order.StatusPending,
		Nonce:       nonce,
		TxHash:      hash,
	}

	err = utils.Tx(ctx, svcCtx.DbClient, func(tx *ent.Tx) error {
		for _, item := range data {
			if item.Status != grid.StatusBought {
				continue
			}

			_, err := model.NewGridModel(tx.Grid).DeleteByGuid(ctx, item.GUID)
			if err != nil {
				return err
			}
		}

		_, err = model.NewOrderModel(tx.Order).Save(ctx, orderArgs)
		return err
	})
	if err != nil {
		logger.Errorf("[ClosePosition] 清仓代币 - 保存订单失败, order: %+v, %v", orderArgs, err)
	}
}

func FetchTokenCandles(ctx context.Context, svcCtx *svc.ServiceContext, token string, to time.Time, period string, limit int) ([]charts.Ohlc, error) {
	if svcCtx.Config.Datapi == "okx" {
		return svcCtx.OkxClient.FetchTokenCandles(ctx, token, to, period, limit)
	}
	return svcCtx.GmgnClient.FetchTokenCandles(ctx, token, to, period, limit)
}

func GetStrategyDetailsText(ctx context.Context, svcCtx *svc.ServiceContext, record *ent.Strategy) string {
	// 生成网格列表
	gridPrices, err := utils.GenerateGrid(record.LowerPriceBound, record.UpperPriceBound, record.TakeProfitRatio.Div(decimal.NewFromInt(100)))
	if err != nil {
		logger.Debugf("[GetStrategyDetailsText] 生成网格列表失败, low: %v, up: %v, takeProfitRatio: %v, %v",
			record.LowerPriceBound, record.UpperPriceBound, record.TakeProfitRatio, err)
	}

	// 获取网格数据
	gridMap := make(map[int]*ent.Grid)
	gridRecords, err := svcCtx.GridModel.FindByStrategyId(ctx, record.GUID)
	if err == nil {
		for _, gridRecord := range gridRecords {
			gridMap[gridRecord.GridNumber] = gridRecord
		}
	} else {
		logger.Debugf("[GetStrategyDetailsText] 查找网格列表失败, strategy: %v, %v", record.GUID, err)
	}

	// 获取当前价格
	var currentPrice decimal.Decimal
	ohlcs, err := FetchTokenCandles(ctx, svcCtx, record.Token, time.Now(), "1m", 30)
	if err != nil {
		logger.Warnf("[GetStrategyDetailsText] 获取 ohlcs 数据失败, token: %s, %v", record.Token, err)
	}

	lastUpdateTime := time.Now()
	if len(ohlcs) > 0 {
		currentPrice = ohlcs[len(ohlcs)-1].Close
		lastUpdateTime = ohlcs[len(ohlcs)-1].Time
	}

	// 查询已实现利润
	var reallzedProfit decimal.Decimal
	if record.FirstOrderId != nil {
		reallzedProfit, err = svcCtx.OrderModel.TotalProfit(ctx, record.GUID, *record.FirstOrderId)
		if err != nil {
			logger.Warnf("[GetStrategyDetailsText] 获取已实现盈亏失败, strategy: %s, %v", record.GUID, err)
		}
	}

	// 计算未实现利润
	var unreallzed decimal.Decimal
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		unreallzed = unreallzed.Add(item.Quantity.Mul(currentPrice).Sub(item.Amount))
	}

	// 计算交易量
	lastKlineVolume := decimal.Zero
	if len(ohlcs) > 0 {
		lastKlineVolume = ohlcs[len(ohlcs)-1].Volume
		if len(ohlcs) > 1 && lastKlineVolume.LessThan(ohlcs[len(ohlcs)-2].Volume) {
			lastKlineVolume = ohlcs[len(ohlcs)-2].Volume
		}
	}

	accumulator := func(agg decimal.Decimal, item charts.Ohlc, idx int) decimal.Decimal {
		return agg.Add(item.Volume)
	}
	fiveKlineVolume := lo.Reduce(lo.Slice(ohlcs, len(ohlcs)-5, len(ohlcs)), accumulator, decimal.Zero)
	tenKlineVolume := lo.Reduce(lo.Slice(ohlcs, len(ohlcs)-10, len(ohlcs)), accumulator, decimal.Zero)
	twentyKlineVolume := lo.Reduce(lo.Slice(ohlcs, len(ohlcs)-20, len(ohlcs)), accumulator, decimal.Zero)
	thirtyKlineVolume := lo.Reduce(lo.Slice(ohlcs, len(ohlcs)-30, len(ohlcs)), accumulator, decimal.Zero)

	// 计算防瀑布跌幅
	dropText := ""
	if record.DropOn && len(ohlcs) > 0 && record.CandlesToCheck > 0 {
		candles := lo.Slice(ohlcs, len(ohlcs)-record.CandlesToCheck, len(ohlcs))
		drop := candles[0].Open.Sub(currentPrice).Div(candles[0].Open).Mul(decimal.NewFromInt(100))
		dropText = fmt.Sprintf("📉 最近%d分钟最大跌幅: %s%%\n", record.CandlesToCheck, drop.Truncate(2))
	}

	// 生成网格详情
	chainId := svcCtx.Config.Chain.Id
	text := fmt.Sprintf("%s 网格机器人 | *%s* 策略详情", utils.GetNetworkName(chainId), strings.TrimRight(record.Symbol, "\u0000"))
	text = text + fmt.Sprintf("\n\n[OKX](%s) | [GMGN](%s) | [DEX Scanner](%s)",
		utils.GetOkxTokenLink(chainId, record.Token), utils.GetGmgnTokenLink(chainId, record.Token), utils.GetDexscreenerTokenLink(chainId, record.Token))
	text = text + fmt.Sprintf("\n\n📈 价格区间: *$%s ~ $%s*\n", record.LowerPriceBound.String(), record.UpperPriceBound.String())
	text = text + fmt.Sprintf("⚙️ 单格投入: *%s 𝗨𝗦𝗗𝗖*\n", record.InitialOrderSize.String())
	text = text + fmt.Sprintf("🔄 网格详情: *%d格 (%s%% 止盈)*\n", len(gridPrices), record.TakeProfitRatio.String())
	text = text + fmt.Sprintf("💵 总利润: %s\n", reallzedProfit.Add(unreallzed).Truncate(2))
	text = text + fmt.Sprintf("✅ 已实现利润: %s\n", reallzedProfit.Truncate(2))
	text = text + fmt.Sprintf("❓ 未实现利润: %s\n", unreallzed.Truncate(2))
	text = text + fmt.Sprintf("💰 最近交易量: %s\n", humanize.Comma(lastKlineVolume.IntPart()))
	text = text + fmt.Sprintf("💰 最近5分钟交易量: %s\n", humanize.Comma(fiveKlineVolume.IntPart()))
	text = text + fmt.Sprintf("💰 最近10分钟交易量: %s\n", humanize.Comma(tenKlineVolume.IntPart()))
	text = text + fmt.Sprintf("💰 最近20分钟交易量: %s\n", humanize.Comma(twentyKlineVolume.IntPart()))
	text = text + fmt.Sprintf("💰 最近30分钟交易量: %s\n", humanize.Comma(thirtyKlineVolume.IntPart()))
	if dropText != "" {
		text = text + dropText
	}
	text = text + "\n⚪️ 待买入 │ 🟡 买入中 │ 🟢 已买入 | 🔴 卖出中\n\n"

	// 计算分割位置
	splitPos := 0
	for idx, gridPrice := range gridPrices {
		if currentPrice.LessThan(gridPrice) {
			break
		}
		splitPos = idx + 1
	}

	// 生成网格标签
	var gridLabels []string
	for idx, gridPrice := range gridPrices {
		status := "⚪️"
		grideRecord, ok := gridMap[idx]
		if ok {
			switch grideRecord.Status {
			case grid.StatusBuying:
				status = "🟡"
			case grid.StatusBought:
				status = "🟢"
			case grid.StatusSelling:
				status = "🔴"
			}
		}

		item := fmt.Sprintf("➖\\[ *%d* ] %s %v", idx, format.Price(gridPrice, 5), status)
		if idx == 0 {
			item += " *(网格底部)*"
		}
		if idx == len(gridPrices)-1 {
			item += " *(网格顶部)*"
		}
		gridLabels = append(gridLabels, item)
	}

	// 截断网格列表
	var maxItems = 10
	var part1, part2 []string
	if splitPos == 0 {
		part2 = gridLabels
	} else if splitPos == len(gridLabels) {
		part1 = gridLabels
	} else {
		part1 = lo.Slice(gridLabels, 0, splitPos)
		part2 = lo.Slice(gridLabels, splitPos, len(gridLabels))
	}

	gridLabels = make([]string, 0, len(gridLabels))
	currentPriceLabel := format.Price(currentPrice, 5)
	if len(part1) > 0 {
		if len(part1) <= maxItems {
			// 处理需要裁剪网格数量的情况
			gridLabels = append(gridLabels, part1...)
			if len(part2) == 0 {
				gridLabels = append(gridLabels, fmt.Sprintf("➖[💵] *当前价格*: $*%s*", currentPriceLabel))
			}
		} else {
			// 处理无需裁剪网格数量的情况
			if len(part2) > 0 {
				// 省略符插入两段之间
				gridLabels = append([]string{part1[0], "➖   ... (省略中间网格)"}, lo.Slice(part1, len(part1)-maxItems, len(part1))...)
			} else {
				// 处理全部网格低于当前价格的情况
				maxItems = maxItems * 2
				if len(part1)-maxItems <= 0 {
					gridLabels = append(gridLabels, part1...)
				} else {
					gridLabels = append([]string{part1[0], "➖   ... (省略中间网格)"}, lo.Slice(part1, len(part1)-maxItems, len(part1))...)
				}

				gridLabels = append(gridLabels, fmt.Sprintf("➖[💵] *当前价格*: $*%s*", currentPriceLabel))
			}

		}
	}

	if len(part1) > 0 && len(part2) > 0 {
		gridLabels = append(gridLabels, fmt.Sprintf("➖[💵] *当前价格*: $*%s*", currentPriceLabel))
	}

	if len(part2) > 0 {
		if len(part2) <= maxItems {
			// 处理需要裁剪网格数量的情况
			if len(part1) == 0 {
				gridLabels = append(gridLabels, fmt.Sprintf("➖[💵] *当前价格*: $*%s*", currentPriceLabel))
			}
			gridLabels = append(gridLabels, part2...)
		} else {
			if len(part1) == 0 {
				maxItems = maxItems * 2
			}
			if len(part1) == 0 {
				gridLabels = append(gridLabels, fmt.Sprintf("➖[💵] *当前价格*: $*%s*", currentPriceLabel))
			}

			gridLabels = append(gridLabels, lo.Slice(part2, 0, maxItems)...)
			if len(part2) > maxItems {
				if len(part2) == maxItems+1 {
					gridLabels = append(gridLabels, part2[len(part2)-1])
				} else {
					gridLabels = append(gridLabels, "➖   ... (省略中间网格)", part2[len(part2)-1])
				}
			}
		}
	}

	if len(gridLabels) == 0 {
		gridLabels = append(gridLabels, fmt.Sprintf("➖[💵] *当前价格*: $*%s*", currentPriceLabel))
	}
	slices.Reverse(gridLabels)

	// 生成网格详情
	text = text + strings.Join(gridLabels, "\n")
	text = text + fmt.Sprintf("\n\n🕒 更新时间: [%s]\n\n⚠️ 重要提示:\n▸ *停止策略会清空之前的网格记录!*", utils.FormaTime(lastUpdateTime))

	return text
}

func DisplayStrategyHomeMenu(ctx context.Context, svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI, userId int64, update tgbotapi.Update, page int) error {
	if page < 1 {
		return nil
	}

	// 查询策略列表
	const limit = 10
	offset := (page - 1) * limit
	data, total, err := svcCtx.StrategyModel.FindByUserId(ctx, userId, offset, limit)
	if err != nil {
		return err
	}

	totalPage := total / limit
	if total%limit != 0 {
		totalPage += 1
	}

	if page > totalPage {
		page = totalPage
		offset := (page - 1) * limit
		data, total, err = svcCtx.StrategyModel.FindByUserId(ctx, userId, offset, limit)
		if err != nil {
			return err
		}
	}

	// 生成策略列表
	var strategyButtons [][]tgbotapi.InlineKeyboardButton
	for _, item := range data {
		status := "🟢"
		if item.Status != strategy.StatusActive {
			status = "🔴"
		} else if !item.EnableAutoBuy {
			status = "⏸️"
		}
		text := fmt.Sprintf("%s %s | 单笔: %vU | 止盈: %v%%",
			status, strings.TrimRight(item.Symbol, "\u0000"), item.InitialOrderSize.String(), item.TakeProfitRatio.String())
		strategyButtons = append(strategyButtons, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(text, StrategyDetailsHandler{}.FormatPath(item.GUID)),
		})
	}

	// 多页翻页功能
	var pageButtons []tgbotapi.InlineKeyboardButton
	if total > limit {
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
			tgbotapi.NewInlineKeyboardButtonData("⬅️ 上一页", StrategyHomeHandler{}.FormatPath(previousPage)),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d/%d", page, totalPage), "/strategy/my/page/0"),
			tgbotapi.NewInlineKeyboardButtonData("➡️ 下一页", StrategyHomeHandler{}.FormatPath(nextPage)),
		}
	}

	rows := strategyButtons
	if len(pageButtons) > 0 {
		rows = append(rows, pageButtons)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ 返回", "/home"),
		tgbotapi.NewInlineKeyboardButtonData("➕ 新建策略", NewStrategyHandler{}.FormatPath()),
	))
	markup := tgbotapi.NewInlineKeyboardMarkup(rows...)

	chainId := svcCtx.Config.Chain.Id
	text := fmt.Sprintf("%s 网格机器人 | 我的策略\n\n⏳ 7x24小时自动化交易\n🔥 市场震荡行情的最佳解决方案\n\n*[核心优势]*\n✓ 突破传统低买高卖模式\n✓ 震荡行情中收益最大化\n\n*[适用场景]*\n🔸 横盘震荡行情\n🔸 主流币/稳定币交易对",
		utils.GetNetworkName(chainId))
	_, err = utils.ReplyMessage(botApi, update, text, markup)
	if err != nil {
		logger.Debugf("[TeleBot] 处理策略主页失败, %v", err)
	}
	return nil
}

func DisplayStrategyDetailsMenu(ctx context.Context, svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI, userId int64, update tgbotapi.Update, record *ent.Strategy) error {
	status := "🟢 策略运行中"
	if record.Status != strategy.StatusActive {
		status = "🔴 策略已停止"
	} else if !record.EnableAutoBuy {
		status = "⏸️ 策略运行中"
	}

	text := GetStrategyDetailsText(ctx, svcCtx, record)

	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 刷新界面", StrategyDetailsHandler{}.FormatPath(record.GUID)),
			tgbotapi.NewInlineKeyboardButtonData("💎 一键清仓", ClosePositionyHandler{}.FormatPath(record.GUID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(status, StrategySwitchHandler{}.FormatPath(record.GUID)),
			tgbotapi.NewInlineKeyboardButtonData("🗒 交易记录", StrategyTradesHandler{}.FormatPath(record.GUID, 1)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ 策略配置", StrategySettingsHandler{}.FormatPath(record.GUID, nil)),
			tgbotapi.NewInlineKeyboardButtonData("🗑 删除策略", DeleteStrategyHandler{}.FormatPath(record.GUID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ 返回上级", StrategyHomeHandler{}.FormatPath(1)),
			tgbotapi.NewInlineKeyboardButtonData("⏪ 返回主页", "/home"),
		),
	)
	_, err := utils.ReplyMessage(botApi, update, text, markup)
	if err != nil {
		logger.Debugf("[TeleBot] 处理策略详情失败, %v", err)
	}
	return nil
}

func DisplayStrategSettingsMenu(svcCtx *svc.ServiceContext, botApi *tgbotapi.BotAPI, update tgbotapi.Update, record *ent.Strategy) error {
	lastKlineVolume, fiveKlineVolume := "-", "-"
	if record.LastKlineVolume != nil && !record.LastKlineVolume.IsZero() {
		lastKlineVolume = humanize.Comma(record.LastKlineVolume.IntPart())
	}
	if record.FiveKlineVolume != nil && !record.FiveKlineVolume.IsZero() {
		fiveKlineVolume = humanize.Comma(record.FiveKlineVolume.IntPart())
	}

	maxGridLimit := "-"
	if record.MaxGridLimit != nil && *record.MaxGridLimit > 0 {
		maxGridLimit = strconv.Itoa(*record.MaxGridLimit)
	}

	stopLossExit := "-"
	if record.StopLossExit != nil && !record.StopLossExit.IsZero() {
		stopLossExit = "-" + record.StopLossExit.Truncate(2).String() + "U"
	}

	takeProfitExit := "-"
	if record.TakeProfitExit != nil && !record.TakeProfitExit.IsZero() {
		takeProfitExit = "+" + record.TakeProfitExit.Truncate(2).String() + "U"
	}

	upperBoundExit := "-"
	if record.UpperBoundExit != nil && record.UpperBoundExit.GreaterThan(decimal.Zero) {
		upperBoundExit = record.UpperBoundExit.String()
	}

	candlesToCheck := "-"
	if record.CandlesToCheck > 0 {
		candlesToCheck = strconv.Itoa(record.CandlesToCheck)
	}

	dropThreshold := "-"
	if record.DropThreshold != nil && record.DropThreshold.GreaterThan(decimal.Zero) {
		dropThreshold = fmt.Sprintf("%v%%", record.DropThreshold.Truncate(2))
	}

	globalTakeProfitRatio := "-"
	if record.GlobalTakeProfitRatio != nil && !record.GlobalTakeProfitRatio.IsZero() {
		globalTakeProfitRatio = "+" + record.GlobalTakeProfitRatio.Mul(decimal.NewFromInt(100)).Truncate(2).String() + "%"
	}

	h := StrategySettingsHandler{}
	chainId := svcCtx.Config.Chain.Id
	text := fmt.Sprintf("%s 网格机器人 | *%s* 编辑策略\n\n`%s`\n\n`「调整设置, 优化您的交易体验」`",
		utils.GetNetworkName(chainId), strings.TrimRight(record.Symbol, "\u0000"), record.Token)
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				lo.If(record.EnableAutoBuy, "🟢 自动买入打开").Else("🔴 自动买入关闭"), h.FormatPath(record.GUID, &SettingsOptionEnableAutoBuy)),
			tgbotapi.NewInlineKeyboardButtonData(
				lo.If(record.EnableAutoSell, "🟢 自动止盈打开").Else("🔴 自动止盈关闭"), h.FormatPath(record.GUID, &SettingsOptionEnableAutoSell)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				lo.If(record.EnablePushNotification, "🟢 消息推送打开").Else("🔴 消息推送关闭"), h.FormatPath(record.GUID, &SettingsOptionEnablePushNotification)),
			tgbotapi.NewInlineKeyboardButtonData(
				lo.If(record.EnableAutoExit, "🟢 自动清仓打开").Else("🔴 自动清仓关闭"), h.FormatPath(record.GUID, &SettingsOptionEnableAutoClear)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("止盈金额 %s", takeProfitExit), h.FormatPath(record.GUID, &SettingsOptionTakeProfitExit)),
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("止损金额 %s", stopLossExit), h.FormatPath(record.GUID, &SettingsOptionStopLossExit)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("总体止盈率 %v", globalTakeProfitRatio), h.FormatPath(record.GUID, &SettingsOptionGlobalTakeProfitRatio)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("离场目标价格: %v", upperBoundExit), h.FormatPath(record.GUID, &SettingsOptionUpperBoundExit)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🟰 止盈 %s%%", record.TakeProfitRatio), h.FormatPath(record.GUID, &SettingsOptionTakeProfitRatio)),
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("🟰 每格 %vU", record.InitialOrderSize), h.FormatPath(record.GUID, &SettingsOptionOrderSize)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("♾️ 网格上限 %s", maxGridLimit), h.FormatPath(record.GUID, &SettingsOptionMaxGridLimit)),
			tgbotapi.NewInlineKeyboardButtonData(
				lo.If(record.DynamicStopLoss, "🟢 动态止损打开").Else("🔴 动态止损关闭"), h.FormatPath(record.GUID, &SettingsOptionDynamicStopLoss)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(lo.If(record.DropOn, "🟢 防瀑布打开").Else("🔴 防瀑布关闭"), h.FormatPath(record.GUID, &SettingsOptionDropOn)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("K线根数: %s", candlesToCheck), h.FormatPath(record.GUID, &SettingsOptionCandlesToCheck)),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("跌幅阈值: %s", dropThreshold), h.FormatPath(record.GUID, &SettingsOptionDropThreshold)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("⬆️ 价格上限 %v", record.UpperPriceBound), h.FormatPath(record.GUID, &SettingsOptionUpperPriceBound)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("⬇️ 价格下限 %v", record.LowerPriceBound), h.FormatPath(record.GUID, &SettingsOptionLowerPriceBound)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("➖ 最近交易量 %v", lastKlineVolume), h.FormatPath(record.GUID, &SettingsOptionLastKlineVolume)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("➖ 5分钟交易量 %v", fiveKlineVolume), h.FormatPath(record.GUID, &SettingsOptionFiveKlineVolume)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ 返回上级", StrategyDetailsHandler{}.FormatPath(record.GUID)),
			tgbotapi.NewInlineKeyboardButtonData("⏪ 返回主页", "/home"),
		),
	)
	_, err := utils.ReplyMessage(botApi, update, text, markup)
	return err
}
