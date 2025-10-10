package strategy

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/fachebot/evm-grid-bot/internal/charts"
	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/grid"
	"github.com/fachebot/evm-grid-bot/internal/ent/order"
	entstrategy "github.com/fachebot/evm-grid-bot/internal/ent/strategy"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/model"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/swap"
	"github.com/fachebot/evm-grid-bot/internal/utils"
	"github.com/fachebot/evm-grid-bot/internal/utils/evm"
	"github.com/fachebot/evm-grid-bot/internal/utils/format"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

type GridStrategy struct {
	svcCtx       *svc.ServiceContext
	strategyId   string
	tokenAddress string
}

func NewGridStrategy(svcCtx *svc.ServiceContext, s *ent.Strategy) *GridStrategy {
	return &GridStrategy{
		svcCtx:       svcCtx,
		strategyId:   s.GUID,
		tokenAddress: s.Token,
	}
}

func calculateTotalProfit(ctx context.Context, svcCtx *svc.ServiceContext, strategyRecord *ent.Strategy, gridRecords []*ent.Grid, latestPrice decimal.Decimal) (decimal.Decimal, error) {
	// 获取累计盈利
	var err error
	var realizedProfit decimal.Decimal
	if strategyRecord.FirstOrderId != nil {
		realizedProfit, err = svcCtx.OrderModel.TotalProfit(ctx, strategyRecord.GUID, *strategyRecord.FirstOrderId)
		if err != nil {
			return decimal.Zero, nil
		}
	}

	// 计算未实现盈利
	unreallzed := decimal.Zero
	uiTotalAmount := decimal.Zero
	uiTotalQuantity := decimal.Zero
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		uiTotalAmount = uiTotalAmount.Add(item.Amount)
		uiTotalQuantity = uiTotalQuantity.Add(item.Quantity)
		unreallzed = unreallzed.Add(item.Quantity.Mul(latestPrice).Sub(item.Amount))
	}

	return realizedProfit.Add(unreallzed), nil
}

func (s *GridStrategy) ID() string {
	return s.strategyId
}

func (s *GridStrategy) TokenAddress() string {
	return s.tokenAddress
}

func (s *GridStrategy) OnTick(ctx context.Context, ohlcs []charts.Ohlc) error {
	// 获取策略信息
	strategyRecord, err := s.svcCtx.StrategyModel.FindByGUID(ctx, s.strategyId)
	if err != nil {
		logger.Errorf("[GridStrategy] 查询策略记录失败, strategy: %v, %v", s.strategyId, err)
		return err
	}

	if strategyRecord.Status != entstrategy.StatusActive {
		logger.Debugf("[GridStrategy] 策略已停止, strategy: %v", s.strategyId)
		return nil
	}

	// 获取网格列表
	gridRecords, err := s.svcCtx.GridModel.FindByStrategyId(ctx, s.strategyId)
	if err != nil {
		logger.Errorf("[GridStrategy] 查询网格列表失败, strategy: %v, %v", s.strategyId, err)
		return err
	}
	gridMapper := make(map[int]*ent.Grid)
	for _, item := range gridRecords {
		gridMapper[item.GridNumber] = item
	}

	// 处理瀑布下跌
	success, err := s.handleWaterfallDrop(ctx, strategyRecord, gridRecords, ohlcs)
	if success {
		return nil
	}
	if err != nil {
		return err
	}

	// 突破退场价格
	latestPrice := ohlcs[len(ohlcs)-1].Close
	success, err = s.handleUpperBoundExit(ctx, strategyRecord, gridRecords, latestPrice)
	if success {
		return nil
	}
	if err != nil {
		return err
	}

	// 获取累计盈利
	totalProfit, err := calculateTotalProfit(ctx, s.svcCtx, strategyRecord, gridRecords, latestPrice)
	if err != nil {
		logger.Errorf("[GridStrategy] 计算策略总利润失败, strategy: %v, %v", s.strategyId, err)
		return err
	}

	// 处理全局止盈
	success, err = s.handleGlobalTakeProfit(ctx, strategyRecord, gridRecords, totalProfit, latestPrice)
	if success {
		return nil
	}
	if err != nil {
		return err
	}

	// 达到盈利目标
	success, err = s.handleTakeProfitAtTarget(ctx, strategyRecord, gridRecords, totalProfit, latestPrice)
	if success {
		return nil
	}
	if err != nil {
		return err
	}

	// 达到亏损阈值
	success, err = s.handleStopLossAtThreshold(ctx, strategyRecord, gridRecords, totalProfit, latestPrice)
	if success {
		return nil
	}
	if err != nil {
		return err
	}

	// 计算网格止盈
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}

		// 处理网格止盈
		s.handleTakeProfit(ctx, latestPrice, strategyRecord, item)
	}

	// 生成网格列表
	gridList, err := utils.GenerateGrid(
		strategyRecord.LowerPriceBound, strategyRecord.UpperPriceBound, strategyRecord.TakeProfitRatio.Div(decimal.NewFromInt(100)))
	if err != nil {
		logger.Errorf("[GridStrategy] 生成网格列表失败, strategy: %v, lowerPriceBound: %v, upperPriceBound: %v, takeProfitRatio: %v, %v",
			s.strategyId, strategyRecord.LowerPriceBound, strategyRecord.UpperPriceBound, strategyRecord.TakeProfitRatio, err)
		return err
	}

	// 计算网格编号
	gridNumber, ok := utils.CalculateGridPosition(gridList, latestPrice)
	if !ok {
		gridNumber = math.MaxInt
	}

	// 更新交易趋势
	gridTrend := decodeGridTrend(strategyRecord.GridTrend)
	if len(gridTrend) == 0 {
		gridTrend = updateGridTrend(gridTrend, math.MaxInt)
	}
	gridTrend = updateGridTrend(gridTrend, gridNumber)
	err = s.svcCtx.StrategyModel.UpdateGridTrend(ctx, strategyRecord.ID, encodeGridTrend(gridTrend))
	if err != nil {
		logger.Errorf("[GridStrategy] 更新交易趋势失败, strategy: %v, gridTrend: %v, %v",
			s.strategyId, gridTrend, err)
	}

	if !ok {
		logger.Debugf("[GridStrategy] 超出网格范围, strategy: %v, price: %v, lowerPriceBound: %v, upperPriceBound: %v",
			s.strategyId, latestPrice, strategyRecord.LowerPriceBound, strategyRecord.UpperPriceBound)

		if latestPrice.GreaterThan(strategyRecord.UpperPriceBound) {
			s.sendUpperThresholdAlert(ctx, strategyRecord, latestPrice)
		}
		return nil
	}

	// 处理网格交易
	takeProfitRatio := strategyRecord.TakeProfitRatio.Div(decimal.NewFromInt(100))
	exitPrice := strategyRecord.LowerPriceBound.Sub(strategyRecord.LowerPriceBound.Mul(takeProfitRatio))
	if gridNumber == 0 && latestPrice.LessThan(exitPrice) {
		s.handlepriceRangeStopLoss(ctx, strategyRecord, gridRecords, latestPrice)
		return nil
	} else {
		// 处理动态止损
		if strategyRecord.DynamicStopLoss && strategyRecord.MaxGridLimit != nil {
			for _, item := range gridRecords {
				if item.Status != grid.StatusBought {
					continue
				}

				if item.GridNumber-gridNumber < *strategyRecord.MaxGridLimit {
					continue
				}

				s.handleDynamicStopLoss(ctx, strategyRecord, item, gridNumber, latestPrice)
			}
		}

		// 是否下跌趋势
		if len(gridTrend) >= 2 && /*(!isDowntrend(gridTrend)) ||*/ !isMinGridNumber(gridRecords, gridNumber) {
			logger.Debugf("[GridStrategy] 不是下跌趋势, strategy: %v, price: %v, gridTrend: %v", s.strategyId, latestPrice, gridTrend)
			return nil
		}

		// 网格是否使用
		_, ok = gridMapper[gridNumber]
		if ok {
			logger.Debugf("[GridStrategy] 网格已被使用, strategy: %v, price: %v, number: %d, lowerPriceBound: %v, upperPriceBound: %v",
				s.strategyId, latestPrice, gridNumber, strategyRecord.LowerPriceBound, strategyRecord.UpperPriceBound)
			return nil
		}

		s.handleGridBuy(ctx, strategyRecord, ohlcs, gridRecords, gridNumber, gridList[gridNumber])
	}

	return nil
}

func (s *GridStrategy) sendUpperThresholdAlert(ctx context.Context, strategyRecord *ent.Strategy, latestPrice decimal.Decimal) {
	if !strategyRecord.EnablePushNotification {
		return
	}
	if strategyRecord.LastUpperThresholdAlertTime != nil {
		return
	}

	percentage := latestPrice.Sub(strategyRecord.UpperPriceBound).Div(strategyRecord.UpperPriceBound).Mul(decimal.NewFromInt(100))
	text := "🚨*%s* 突破价格上限!\n\n`%s`\n\n💥 当前价格: %s (上限设定: %s)\n📈 已突破上限: %s%%\n\n✅ 止盈功能仍正常运行中!\n⚠️ 系统已自动暂停新买入订单!"
	text = fmt.Sprintf(text, strategyRecord.Symbol, strategyRecord.Token, format.Price(latestPrice, 5), strategyRecord.UpperPriceBound, percentage.Truncate(2))

	_, err := utils.SendMessage(s.svcCtx.BotApi, strategyRecord.UserId, text)
	if err != nil {
		logger.Warnf("[GridStrategy] 发送电报通知失败, userId: %d, text: %s, %v", strategyRecord.UserId, text, err)
		return
	}

	// 更新最后一次警报时间
	err = s.svcCtx.StrategyModel.UpdateLastUpperThresholdAlertTime(ctx, strategyRecord.ID, time.Now())
	if err != nil {
		logger.Errorf("[GridStrategy] 更新最后一次警报时间失败, strategy: %s, %v", strategyRecord.GUID, err)
		return
	}
}

func (s *GridStrategy) sendLowerThresholdAlert(ctx context.Context, strategyRecord *ent.Strategy, latestPrice decimal.Decimal) {
	if !strategyRecord.EnablePushNotification {
		return
	}
	if strategyRecord.LastLowerThresholdAlertTime != nil {
		return
	}

	percentage := strategyRecord.LowerPriceBound.Sub(latestPrice).Div(strategyRecord.LowerPriceBound).Mul(decimal.NewFromInt(100))
	text := "🚨*%s* 跌破价格下限!\n\n`%s`\n\n💥 当前价格: %s (下限设定: %s)\n📈 已跌破下限: %s%%\n\n✅ 止盈功能仍正常运行中!\n⚠️ 系统已自动暂停新买入订单!"
	text = fmt.Sprintf(text, strategyRecord.Symbol, strategyRecord.Token, format.Price(latestPrice, 5), strategyRecord.LowerPriceBound, percentage.Truncate(2))

	_, err := utils.SendMessage(s.svcCtx.BotApi, strategyRecord.UserId, text)
	if err != nil {
		logger.Warnf("[GridStrategy] 发送电报通知失败, userId: %d, text: %s, %v", strategyRecord.UserId, text, err)
		return
	}

	// 更新最后一次警报时间
	err = s.svcCtx.StrategyModel.UpdateLastLowerThresholdAlertTime(ctx, strategyRecord.ID, time.Now())
	if err != nil {
		logger.Errorf("[GridStrategy] 更新最后一次警报时间失败, strategy: %s, %v", strategyRecord.GUID, err)
		return
	}
}

func (s *GridStrategy) handleGridBuy(ctx context.Context, strategyRecord *ent.Strategy, ohlcs []charts.Ohlc, gridList []*ent.Grid, gridNumber int, gridPrice decimal.Decimal) {
	if !strategyRecord.EnableAutoBuy {
		return
	}

	// 是否超过上限
	if strategyRecord.MaxGridLimit != nil &&
		*strategyRecord.MaxGridLimit > 0 &&
		len(gridList) >= *strategyRecord.MaxGridLimit {
		return
	}

	guid, err := uuid.NewRandom()
	if err != nil {
		logger.Errorf("[GridStrategy] 生成 GUID 失败, %v", err)
		return
	}

	// 检查交易量条件
	latestPrice := ohlcs[len(ohlcs)-1].Close
	if strategyRecord.LastKlineVolume != nil && !strategyRecord.LastKlineVolume.IsZero() {
		volume := ohlcs[len(ohlcs)-1].Volume
		if len(ohlcs) > 1 && volume.LessThan(ohlcs[len(ohlcs)-2].Volume) {
			volume = ohlcs[len(ohlcs)-2].Volume
		}
		if volume.LessThan(*strategyRecord.LastKlineVolume) {
			logger.Debugf("[GridStrategy] 取消网格买入, 最近K线的交易量不满足要求, volume: %v, require: %v",
				volume, *strategyRecord.LastKlineVolume)
			return
		}
	}

	if strategyRecord.FiveKlineVolume != nil && !strategyRecord.FiveKlineVolume.IsZero() {
		totalVolume := decimal.Zero
		for i := len(ohlcs) - 1; i >= 0 && i >= len(ohlcs)-5; i-- {
			totalVolume = totalVolume.Add(ohlcs[i].Volume)
		}
		if totalVolume.LessThan(*strategyRecord.FiveKlineVolume) {
			logger.Debugf("[GridStrategy] 取消网格买入, 最近5分钟的交易量不满足要求, volume: %v, require: %v",
				totalVolume, *strategyRecord.FiveKlineVolume)
			return
		}
	}

	tokenMeta, err := s.svcCtx.TokenMetaCache.GetTokenMeta(ctx, strategyRecord.Token)
	if err != nil {
		logger.Errorf("[GridStrategy] 获取Token元信息失败, token: %s, %v", strategyRecord.Token, err)
		return
	}

	// 获取报价
	amount := evm.FormatUnits(strategyRecord.InitialOrderSize, s.svcCtx.Config.Chain.StablecoinDecimals)
	swapService := swap.NewSwapService(s.svcCtx, strategyRecord.UserId)
	tx, err := swapService.Quote(ctx, s.svcCtx.Config.Chain.StablecoinCA, strategyRecord.Token, amount)
	if err != nil {
		logger.Errorf("[GridStrategy] 获取报价失败, in: %s, out: %s, amount: %s, %v",
			s.svcCtx.Config.Chain.StablecoinSymbol, strategyRecord.Symbol, strategyRecord.InitialOrderSize, err)
		return
	}

	bottomPrice := gridPrice
	uiOutAmount := evm.ParseUnits(tx.OutAmount(), tokenMeta.Decimals)
	quotePrice := strategyRecord.InitialOrderSize.Div(uiOutAmount)
	logger.Debugf("[GridStrategy] 买入网格, token: %s, latestPrice: %s, gridPrice: %s, quotePrice: %s, bottomPrice: %s",
		strategyRecord.Symbol, latestPrice, gridPrice, quotePrice, bottomPrice)

	if quotePrice.GreaterThan(bottomPrice) {
		logger.Debugf("[GridStrategy] 买入网格, 报价高于底价, 取消交易, token: %s, quotePrice: %s, bottomPrice: %s", strategyRecord.Symbol, quotePrice, bottomPrice)
		return
	}

	// 发送交易
	hash, nonce, err := tx.Swap(ctx)
	if err != nil {
		logger.Errorf("[GridStrategy] 买入网格 - 发送交易失败, user: %d, inputAmount: %s, outToken: %s, outAmount: %s, hash: %s, %v",
			strategyRecord.UserId, strategyRecord.InitialOrderSize, strategyRecord.Symbol, uiOutAmount, hash, err)
		return
	}

	logger.Infof("[GridStrategy] 买入网格 - 提交交易成功, user: %d, strategy: %s, gridNumber: %d, hash: %s",
		strategyRecord.UserId, strategyRecord.GUID, gridNumber, hash)

	// 保存网格和订单
	gridArgs := ent.Grid{
		GUID:       guid.String(),
		Account:    tx.Signer(),
		Token:      strategyRecord.Token,
		Symbol:     strategyRecord.Symbol,
		StrategyId: strategyRecord.GUID,
		GridNumber: gridNumber,
		OrderPrice: quotePrice,
		FinalPrice: quotePrice,
		Amount:     strategyRecord.InitialOrderSize,
		Quantity:   uiOutAmount,
		Status:     grid.StatusBuying,
	}

	orderArgs := ent.Order{
		Account:    gridArgs.Account,
		Token:      gridArgs.Token,
		Symbol:     gridArgs.Symbol,
		GridId:     &gridArgs.GUID,
		GridNumber: &gridArgs.GridNumber,
		StrategyId: gridArgs.StrategyId,
		Type:       order.TypeBuy,
		Price:      gridArgs.OrderPrice,
		FinalPrice: gridArgs.FinalPrice,
		InAmount:   gridArgs.Amount,
		OutAmount:  gridArgs.Quantity,
		Status:     order.StatusPending,
		Nonce:      nonce,
		TxHash:     hash,
	}

	err = utils.Tx(ctx, s.svcCtx.DbClient, func(tx *ent.Tx) error {
		_, err = model.NewGridModel(tx.Grid).Save(ctx, gridArgs)
		if err != nil {
			return err
		}

		_, err = model.NewOrderModel(tx.Order).Save(ctx, orderArgs)
		return nil
	})
	if err != nil {
		logger.Errorf("[GridStrategy] 买入网格 - 保存网格和订单失败, grid: %+v, order: %+v, %v", gridArgs, orderArgs, err)
		return
	}
}

func (s *GridStrategy) handleTakeProfit(ctx context.Context, latestPrice decimal.Decimal, strategyRecord *ent.Strategy, gridRecord *ent.Grid) {
	if !strategyRecord.EnableAutoSell {
		return
	}
	if gridRecord.Status != grid.StatusBought {
		return
	}

	// 计算利润
	profit := gridRecord.FinalPrice.Mul(strategyRecord.TakeProfitRatio.Div(decimal.NewFromInt(100)))
	if latestPrice.LessThan(gridRecord.FinalPrice.Add(profit)) {
		return
	}

	// 卖出代币
	bottomPrice := gridRecord.FinalPrice.Add(profit)
	orderArgs, err := SellToken(ctx, s.svcCtx, strategyRecord, "止盈网格", &gridRecord.Quantity, &bottomPrice, false)
	if err != nil {
		return
	}
	orderArgs.GridId = &gridRecord.GUID
	orderArgs.GridNumber = &gridRecord.GridNumber
	orderArgs.GridBuyCost = &gridRecord.Amount

	// 更新数据状态
	err = utils.Tx(ctx, s.svcCtx.DbClient, func(tx *ent.Tx) error {
		err = model.NewGridModel(tx.Grid).SetSellingStatus(ctx, gridRecord.GUID)
		if err != nil {
			return err
		}

		_, err = model.NewOrderModel(tx.Order).Save(ctx, orderArgs)
		if err != nil {
			return err
		}

		err = model.NewStrategyModel(tx.Strategy).ClearLastLowerThresholdAlertTime(ctx, strategyRecord.ID)
		if err != nil {
			return err
		}

		err = model.NewStrategyModel(tx.Strategy).ClearLastUpperThresholdAlertTime(ctx, strategyRecord.ID)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logger.Errorf("[GridStrategy] 止盈网格 - 更新网格和订单失败, strategy: %s, gridNumber: %d, gridGuid: %s, order: %+v, %v",
			strategyRecord.GUID, gridRecord.GridNumber, gridRecord.GUID, orderArgs, err)
		return
	}

	// 更新网格状态
	gridRecord.Status = grid.StatusSelling
}

func (s *GridStrategy) handlepriceRangeStopLoss(ctx context.Context, strategyRecord *ent.Strategy, gridRecords []*ent.Grid, latestPrice decimal.Decimal) {
	// 计算总仓位
	uiTotalAmount := decimal.Zero
	uiTotalQuantity := decimal.Zero
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		uiTotalAmount = uiTotalAmount.Add(item.Amount)
		uiTotalQuantity = uiTotalQuantity.Add(item.Quantity)
	}
	if len(gridRecords) == 0 || uiTotalQuantity.LessThanOrEqual(decimal.Zero) {
		logger.Debugf("[GridStrategy] 无可清仓网格, strategy: %v, token: %s, totalAmount: %s", s.strategyId, strategyRecord.Symbol, uiTotalQuantity)
		return
	}

	// 发送警报通知
	s.sendLowerThresholdAlert(ctx, strategyRecord, latestPrice)

	// 是否开启清仓
	if !strategyRecord.EnableAutoExit {
		return
	}

	logger.Infof("[GridStrategy] 跌破清仓止损, strategy: %v, token: %s, price: %v, quantity: %v",
		s.strategyId, strategyRecord.Symbol, latestPrice, uiTotalQuantity)

	// 卖出所有代币
	minSellPrice := latestPrice.Sub(latestPrice.Mul(decimal.NewFromFloat(0.01)))
	orderArgs, err := SellToken(ctx, s.svcCtx, strategyRecord, "跌破清仓", nil, &minSellPrice, true)
	if err != nil {
		return
	}
	orderArgs.GridBuyCost = &uiTotalAmount

	// 更新数据状态
	err = utils.Tx(ctx, s.svcCtx.DbClient, func(tx *ent.Tx) error {
		_, err := model.NewGridModel(tx.Grid).DeleteByStrategyId(ctx, strategyRecord.GUID)
		if err != nil {
			return err
		}

		_, err = model.NewOrderModel(tx.Order).Save(ctx, orderArgs)
		if err != nil {
			return err
		}

		err = model.NewStrategyModel(tx.Strategy).UpdateFirstOrderId(ctx, strategyRecord.ID, nil)
		if err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatusByGuid(ctx, strategyRecord.GUID, entstrategy.StatusInactive)
	})
	if err != nil {
		logger.Errorf("[GridStrategy] 跌破清仓止损 - 保存订单失败, order: %+v, %v", orderArgs, err)
	}

	// 更新网格状态
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		item.Status = grid.StatusSelling
	}

	// 停止策略运行
	s.svcCtx.Engine.StopStrategy(strategyRecord.GUID)
}

func (s *GridStrategy) handleWaterfallDrop(ctx context.Context, strategyRecord *ent.Strategy, gridRecords []*ent.Grid, ohlcs []charts.Ohlc) (bool, error) {
	if !strategyRecord.DropOn {
		return false, nil
	}

	if strategyRecord.CandlesToCheck <= 0 {
		return false, nil
	}

	if strategyRecord.DropThreshold == nil || strategyRecord.DropThreshold.LessThanOrEqual(decimal.Zero) {
		return false, nil
	}

	ohlcs = lo.Slice(ohlcs, len(ohlcs)-strategyRecord.CandlesToCheck, len(ohlcs))
	if len(ohlcs) <= 0 {
		return false, nil
	}

	// 是否达到阈值
	latestPrice := ohlcs[len(ohlcs)-1].Close
	drop := ohlcs[0].Open.Sub(latestPrice).Div(ohlcs[0].Open).Mul(decimal.NewFromInt(100))
	if drop.LessThan(*strategyRecord.DropThreshold) {
		return false, nil
	}

	if latestPrice.GreaterThan(strategyRecord.UpperPriceBound) {
		return false, nil
	}

	// 计算总仓位
	uiTotalAmount := decimal.Zero
	uiTotalQuantity := decimal.Zero
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		uiTotalAmount = uiTotalAmount.Add(item.Amount)
		uiTotalQuantity = uiTotalQuantity.Add(item.Quantity)
	}

	logger.Infof("[GridStrategy] 触发防瀑布机制, strategy: %v, token: %s, price: %v, drop: %s, dropThreshold: %s",
		s.strategyId, strategyRecord.Symbol, latestPrice, drop, *strategyRecord.DropThreshold)

	// 卖出所有网格
	var orderArgs *ent.Order
	if len(gridRecords) > 0 && uiTotalQuantity.GreaterThan(decimal.Zero) {
		minSellPrice := latestPrice.Sub(latestPrice.Mul(decimal.NewFromFloat(0.01)))
		ord, err := SellToken(ctx, s.svcCtx, strategyRecord, "防瀑布机制", nil, &minSellPrice, true)
		if err != nil {
			return false, err
		}
		orderArgs = &ord
		orderArgs.GridBuyCost = &uiTotalAmount
	}

	// 更新数据状态
	err := utils.Tx(ctx, s.svcCtx.DbClient, func(tx *ent.Tx) error {
		_, err := model.NewGridModel(tx.Grid).DeleteByStrategyId(ctx, strategyRecord.GUID)
		if err != nil {
			return err
		}

		if orderArgs != nil {
			_, err = model.NewOrderModel(tx.Order).Save(ctx, *orderArgs)
			if err != nil {
				return err
			}
		}

		err = model.NewStrategyModel(tx.Strategy).UpdateFirstOrderId(ctx, strategyRecord.ID, nil)
		if err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatusByGuid(ctx, strategyRecord.GUID, entstrategy.StatusInactive)
	})
	if err != nil {
		logger.Errorf("[GridStrategy] 突破退场目标价格 - 更新状态失败, order: %+v, %v", orderArgs, err)
	}

	// 更新网格状态
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		item.Status = grid.StatusSelling
	}

	// 停止策略运行
	s.svcCtx.Engine.StopStrategy(strategyRecord.GUID)

	// 发送电报通知
	text := "🚨*%s* 触发防瀑布机制!\n\n`%s`\n\n🎯 跌幅阈值: %s%%\n💥 当前跌幅: %s%%\n\n✅ 已自动清仓并停止策略!"
	text = fmt.Sprintf(text, strategyRecord.Symbol, strategyRecord.Token, strategyRecord.DropThreshold.Truncate(2), drop.Truncate(2))
	_, err = utils.SendMessage(s.svcCtx.BotApi, strategyRecord.UserId, text)
	if err != nil {
		logger.Warnf("[GridStrategy] 发送电报通知失败, userId: %d, text: %s, %v", strategyRecord.UserId, text, err)
	}

	return true, nil
}

func (s *GridStrategy) handleUpperBoundExit(ctx context.Context, strategyRecord *ent.Strategy, gridRecords []*ent.Grid, latestPrice decimal.Decimal) (bool, error) {
	if !(strategyRecord.UpperBoundExit != nil &&
		strategyRecord.UpperBoundExit.GreaterThan(decimal.Zero) &&
		latestPrice.GreaterThan(*strategyRecord.UpperBoundExit)) {
		return false, nil
	}

	// 计算总仓位
	uiTotalAmount := decimal.Zero
	uiTotalQuantity := decimal.Zero
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		uiTotalAmount = uiTotalAmount.Add(item.Amount)
		uiTotalQuantity = uiTotalQuantity.Add(item.Quantity)
	}

	logger.Infof("[GridStrategy] 突破退场目标价格, strategy: %v, token: %s, price: %v, targetPrice: %s",
		s.strategyId, strategyRecord.Symbol, latestPrice, *strategyRecord.UpperBoundExit)

	// 卖出所有网格
	var orderArgs *ent.Order
	if len(gridRecords) > 0 && uiTotalQuantity.GreaterThan(decimal.Zero) {
		minSellPrice := latestPrice.Sub(latestPrice.Mul(decimal.NewFromFloat(0.01)))
		ord, err := SellToken(ctx, s.svcCtx, strategyRecord, "突破退场目标价格", nil, &minSellPrice, true)
		if err != nil {
			return false, err
		}
		orderArgs = &ord
		orderArgs.GridBuyCost = &uiTotalAmount
	}

	// 更新数据状态
	err := utils.Tx(ctx, s.svcCtx.DbClient, func(tx *ent.Tx) error {
		_, err := model.NewGridModel(tx.Grid).DeleteByStrategyId(ctx, strategyRecord.GUID)
		if err != nil {
			return err
		}

		if orderArgs != nil {
			_, err = model.NewOrderModel(tx.Order).Save(ctx, *orderArgs)
			if err != nil {
				return err
			}
		}

		err = model.NewStrategyModel(tx.Strategy).UpdateFirstOrderId(ctx, strategyRecord.ID, nil)
		if err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatusByGuid(ctx, strategyRecord.GUID, entstrategy.StatusInactive)
	})
	if err != nil {
		logger.Errorf("[GridStrategy] 突破退场目标价格 - 更新状态失败, order: %+v, %v", orderArgs, err)
	}

	// 更新网格状态
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		item.Status = grid.StatusSelling
	}

	// 停止策略运行
	s.svcCtx.Engine.StopStrategy(strategyRecord.GUID)

	// 发送电报通知
	text := "🚨*%s* 突破退场目标价格!\n\n`%s`\n\n🎯 目标价格: %sU\n💥 当前价格: %sU\n\n✅ 已自动清仓并停止策略!"
	text = fmt.Sprintf(text, strategyRecord.Symbol, strategyRecord.Token, *strategyRecord.UpperBoundExit, format.Price(latestPrice, 5))
	_, err = utils.SendMessage(s.svcCtx.BotApi, strategyRecord.UserId, text)
	if err != nil {
		logger.Warnf("[GridStrategy] 发送电报通知失败, userId: %d, text: %s, %v", strategyRecord.UserId, text, err)
	}

	return true, nil
}

func (s *GridStrategy) handleDynamicStopLoss(ctx context.Context, strategyRecord *ent.Strategy, gridRecord *ent.Grid, gridNumber int, latestPrice decimal.Decimal) {
	if !strategyRecord.DynamicStopLoss ||
		strategyRecord.MaxGridLimit == nil ||
		gridRecord.GridNumber-gridNumber < *strategyRecord.MaxGridLimit {
		return
	}

	// 卖出代币
	logger.Infof("[GridStrategy] 动态止损, strategy: %v, token: %s, price: %v, gridNumber: %d, currentGridNumber: %d",
		s.strategyId, strategyRecord.Symbol, latestPrice, gridRecord.GridNumber, gridNumber)
	minSellPrice := latestPrice.Sub(latestPrice.Mul(decimal.NewFromFloat(0.01)))
	orderArgs, err := SellToken(ctx, s.svcCtx, strategyRecord, "动态止损", &gridRecord.Quantity, &minSellPrice, true)
	if err != nil {
		return
	}

	// 更新数据状态
	orderArgs.GridId = &gridRecord.GUID
	orderArgs.GridNumber = &gridRecord.GridNumber
	orderArgs.GridBuyCost = &gridRecord.Amount
	err = utils.Tx(ctx, s.svcCtx.DbClient, func(tx *ent.Tx) error {
		err = model.NewGridModel(tx.Grid).SetSellingStatus(ctx, gridRecord.GUID)
		if err != nil {
			return err
		}

		_, err = model.NewOrderModel(tx.Order).Save(ctx, orderArgs)
		return err
	})
	if err != nil {
		logger.Errorf("[GridStrategy] 动态止损 - 保存订单失败, order: %+v, %v", orderArgs, err)
	}

	// 更新网格状态
	gridRecord.Status = grid.StatusSelling

	// 发送电报通知
	uiOutAmount := orderArgs.OutAmount
	priceDrop := gridRecord.Amount.Sub(uiOutAmount).Div(gridRecord.Amount).Mul(decimal.NewFromInt(100)).Truncate(2)
	text := fmt.Sprintf("🚨*%s* 网格 `#%d` 执行动态止损\n\n当前跌幅: *%v%%*, 预计亏损: *%sU*",
		strategyRecord.Symbol, gridRecord.GridNumber, priceDrop, gridRecord.Amount.Sub(uiOutAmount).Truncate(2))
	_, err = utils.SendMessage(s.svcCtx.BotApi, strategyRecord.UserId, text)
	if err != nil {
		logger.Warnf("[GridStrategy] 发送电报通知失败, userId: %d, text: %s, %v", strategyRecord.UserId, text, err)
	}
}

func (s *GridStrategy) handleGlobalTakeProfit(ctx context.Context, strategyRecord *ent.Strategy, gridRecords []*ent.Grid, totalProfit, latestPrice decimal.Decimal) (bool, error) {
	if !(strategyRecord.GlobalTakeProfitRatio != nil &&
		strategyRecord.GlobalTakeProfitRatio.GreaterThan(decimal.Zero)) {
		return false, nil
	}

	// 计算总仓位
	uiTotalAmount := decimal.Zero
	uiTotalQuantity := decimal.Zero
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		uiTotalAmount = uiTotalAmount.Add(item.Amount)
		uiTotalQuantity = uiTotalQuantity.Add(item.Quantity)
	}

	// 计算盈利比例
	if uiTotalAmount.IsZero() {
		return false, nil
	}
	ratio := totalProfit.Div(uiTotalAmount)
	if ratio.LessThan(*strategyRecord.GlobalTakeProfitRatio) {
		return false, nil
	}

	logger.Infof("[GridStrategy] 触发全局止盈, strategy: %v, token: %s, price: %v, profitRate: %s%%",
		s.strategyId, strategyRecord.Symbol, latestPrice, ratio.Mul(decimal.NewFromInt(100)).Truncate(2))

	// 卖出所有网格
	var orderArgs *ent.Order
	if len(gridRecords) > 0 && uiTotalQuantity.GreaterThan(decimal.Zero) {
		minSellPrice := latestPrice.Sub(latestPrice.Mul(decimal.NewFromFloat(0.01)))
		ord, err := SellToken(ctx, s.svcCtx, strategyRecord, "触发全局止盈", nil, &minSellPrice, true)
		if err != nil {
			return false, err
		}
		orderArgs = &ord
		orderArgs.GridBuyCost = &uiTotalAmount
	}

	// 更新数据状态
	err := utils.Tx(ctx, s.svcCtx.DbClient, func(tx *ent.Tx) error {
		_, err := model.NewGridModel(tx.Grid).DeleteByStrategyId(ctx, strategyRecord.GUID)
		if err != nil {
			return err
		}

		if orderArgs != nil {
			_, err = model.NewOrderModel(tx.Order).Save(ctx, *orderArgs)
			if err != nil {
				return err
			}
		}

		err = model.NewStrategyModel(tx.Strategy).UpdateFirstOrderId(ctx, strategyRecord.ID, nil)
		if err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatusByGuid(ctx, strategyRecord.GUID, entstrategy.StatusInactive)
	})
	if err != nil {
		logger.Errorf("[GridStrategy] 触发全局止盈 - 更新状态失败, order: %+v, %v", orderArgs, err)
	}

	// 更新网格状态
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		item.Status = grid.StatusSelling
	}

	// 停止策略运行
	s.svcCtx.Engine.StopStrategy(strategyRecord.GUID)

	// 发送电报通知
	text := "🚨*%s* 触发全局止盈!\n\n`%s`\n\n🎯 目标涨幅: %s%%\n💥 当前价格: %sU\n\n✅ 已自动清仓并停止策略!"
	text = fmt.Sprintf(text, strategyRecord.Symbol, strategyRecord.Token, ratio.Mul(decimal.NewFromInt(100)).Truncate(2), format.Price(latestPrice, 5))
	_, err = utils.SendMessage(s.svcCtx.BotApi, strategyRecord.UserId, text)
	if err != nil {
		logger.Warnf("[GridStrategy] 发送电报通知失败, userId: %d, text: %s, %v", strategyRecord.UserId, text, err)
	}

	return true, nil
}

func (s *GridStrategy) handleTakeProfitAtTarget(ctx context.Context, strategyRecord *ent.Strategy, gridRecords []*ent.Grid, totalProfit, latestPrice decimal.Decimal) (bool, error) {
	// 计算总仓位
	uiTotalAmount := decimal.Zero
	uiTotalQuantity := decimal.Zero
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		uiTotalAmount = uiTotalAmount.Add(item.Amount)
		uiTotalQuantity = uiTotalQuantity.Add(item.Quantity)
	}

	if strategyRecord.TakeProfitExit == nil ||
		strategyRecord.TakeProfitExit.IsZero() ||
		totalProfit.LessThan(*strategyRecord.TakeProfitExit) {
		return false, nil
	}

	logger.Infof("[GridStrategy] 达到盈利目标, strategy: %v, token: %s, quantity: %v, totalProfit: %s",
		s.strategyId, strategyRecord.Symbol, uiTotalQuantity, totalProfit)

	// 卖出所有网格
	var orderArgs *ent.Order
	if len(gridRecords) > 0 && uiTotalQuantity.GreaterThan(decimal.Zero) {
		minSellPrice := latestPrice.Sub(latestPrice.Mul(decimal.NewFromFloat(0.01)))
		ord, err := SellToken(ctx, s.svcCtx, strategyRecord, "达到盈利目标", nil, &minSellPrice, true)
		if err != nil {
			return false, err
		}
		orderArgs = &ord
		orderArgs.GridBuyCost = &uiTotalAmount
	}

	// 更新数据状态
	err := utils.Tx(ctx, s.svcCtx.DbClient, func(tx *ent.Tx) error {
		_, err := model.NewGridModel(tx.Grid).DeleteByStrategyId(ctx, strategyRecord.GUID)
		if err != nil {
			return err
		}

		if orderArgs != nil {
			_, err = model.NewOrderModel(tx.Order).Save(ctx, *orderArgs)
			if err != nil {
				return err
			}
		}

		err = model.NewStrategyModel(tx.Strategy).UpdateFirstOrderId(ctx, strategyRecord.ID, nil)
		if err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatusByGuid(ctx, strategyRecord.GUID, entstrategy.StatusInactive)
	})
	if err != nil {
		logger.Errorf("[GridStrategy] 达到盈利目标 - 更新状态失败, order: %+v, %v", orderArgs, err)
	}

	// 更新网格状态
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		item.Status = grid.StatusSelling
	}

	// 停止策略运行
	s.svcCtx.Engine.StopStrategy(strategyRecord.GUID)

	// 发送电报通知
	text := "🚨*%s* 达到盈利目标!\n\n`%s`\n\n🎯 盈利目标: %sU\n💥 预计盈利: %sU\n\n✅ 已自动清仓并停止策略!"
	text = fmt.Sprintf(text, strategyRecord.Symbol, strategyRecord.Token, *strategyRecord.TakeProfitExit, totalProfit.Truncate(2))
	_, err = utils.SendMessage(s.svcCtx.BotApi, strategyRecord.UserId, text)
	if err != nil {
		logger.Warnf("[GridStrategy] 发送电报通知失败, userId: %d, text: %s, %v", strategyRecord.UserId, text, err)
	}

	return true, nil
}

func (s *GridStrategy) handleStopLossAtThreshold(ctx context.Context, strategyRecord *ent.Strategy, gridRecords []*ent.Grid, totalProfit, latestPrice decimal.Decimal) (bool, error) {
	if !(strategyRecord.StopLossExit != nil &&
		strategyRecord.StopLossExit.GreaterThan(decimal.Zero) &&
		totalProfit.LessThanOrEqual(strategyRecord.StopLossExit.Neg())) {
		return false, nil
	}

	// 计算总仓位
	uiTotalAmount := decimal.Zero
	uiTotalQuantity := decimal.Zero
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		uiTotalAmount = uiTotalAmount.Add(item.Amount)
		uiTotalQuantity = uiTotalQuantity.Add(item.Quantity)
	}

	logger.Infof("[GridStrategy] 亏损达到预设金额, strategy: %v, token: %s, price: %v, totalProfit: %s, threshold: %s",
		s.strategyId, strategyRecord.Symbol, latestPrice, totalProfit, *strategyRecord.StopLossExit)

	// 卖出所有网格
	var orderArgs *ent.Order
	if len(gridRecords) > 0 && uiTotalQuantity.GreaterThan(decimal.Zero) {
		minSellPrice := latestPrice.Sub(latestPrice.Mul(decimal.NewFromFloat(0.01)))
		ord, err := SellToken(ctx, s.svcCtx, strategyRecord, "亏损达到预设金额", nil, &minSellPrice, true)
		if err != nil {
			return false, err
		}
		orderArgs = &ord
		orderArgs.GridBuyCost = &uiTotalAmount
	}

	// 更新数据状态
	err := utils.Tx(ctx, s.svcCtx.DbClient, func(tx *ent.Tx) error {
		_, err := model.NewGridModel(tx.Grid).DeleteByStrategyId(ctx, strategyRecord.GUID)
		if err != nil {
			return err
		}

		if orderArgs != nil {
			_, err = model.NewOrderModel(tx.Order).Save(ctx, *orderArgs)
			if err != nil {
				return err
			}
		}

		err = model.NewStrategyModel(tx.Strategy).UpdateFirstOrderId(ctx, strategyRecord.ID, nil)
		if err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatusByGuid(ctx, strategyRecord.GUID, entstrategy.StatusInactive)
	})
	if err != nil {
		logger.Errorf("[GridStrategy] 亏损达到预设金额 - 更新状态失败, order: %+v, %v", orderArgs, err)
	}

	// 更新网格状态
	for _, item := range gridRecords {
		if item.Status != grid.StatusBought {
			continue
		}
		item.Status = grid.StatusSelling
	}

	// 停止策略运行
	s.svcCtx.Engine.StopStrategy(strategyRecord.GUID)

	// 发送电报通知
	text := "🚨*%s* 亏损达到预设金额!\n\n`%s`\n\n🎯 亏损金额: %sU\n💥 当前价格: %sU\n\n✅ 已自动清仓并停止策略!"
	text = fmt.Sprintf(text, strategyRecord.Symbol, strategyRecord.Token, totalProfit, format.Price(latestPrice, 5))
	_, err = utils.SendMessage(s.svcCtx.BotApi, strategyRecord.UserId, text)
	if err != nil {
		logger.Warnf("[GridStrategy] 发送电报通知失败, userId: %d, text: %s, %v", strategyRecord.UserId, text, err)
	}

	return true, nil
}
