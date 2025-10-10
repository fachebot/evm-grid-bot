package job

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/grid"
	"github.com/fachebot/evm-grid-bot/internal/ent/order"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/model"
	"github.com/fachebot/evm-grid-bot/internal/strategy"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/utils"
	"github.com/fachebot/evm-grid-bot/internal/utils/evm"
	"github.com/fachebot/evm-grid-bot/internal/utils/format"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type OrderKeeper struct {
	ctx        context.Context
	cancel     context.CancelFunc
	stopChan   chan struct{}
	svcCtx     *svc.ServiceContext
	timeoutTxs map[string]struct{}
}

func NewOrderKeeper(svcCtx *svc.ServiceContext) *OrderKeeper {
	ctx, cancel := context.WithCancel(context.Background())
	return &OrderKeeper{
		ctx:        ctx,
		cancel:     cancel,
		svcCtx:     svcCtx,
		timeoutTxs: map[string]struct{}{},
	}
}

func (keeper *OrderKeeper) Stop() {
	if keeper.stopChan == nil {
		return
	}

	logger.Infof("[OrderKeeper] 准备停止服务")

	keeper.cancel()

	<-keeper.stopChan
	close(keeper.stopChan)
	keeper.stopChan = nil

	logger.Infof("[OrderKeeper] 服务已经停止")
}

func (keeper *OrderKeeper) Start() {
	if keeper.stopChan != nil {
		return
	}

	keeper.stopChan = make(chan struct{})
	logger.Infof("[OrderKeeper] 开始运行服务")
	go keeper.run()
}

func (keeper *OrderKeeper) run() {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			keeper.handlePolling()
			duration := time.Millisecond * 1000
			timer.Reset(duration)
		case <-keeper.ctx.Done():
			keeper.stopChan <- struct{}{}
			return

		}
	}
}

func (keeper *OrderKeeper) sendNotification(ord *ent.Order, text string, force bool) {
	w, err := keeper.svcCtx.WalletModel.FindByAccount(keeper.ctx, ord.Account)
	if err != nil {
		logger.Errorf("[OrderKeeper] 查询钱包信息失败, account: %s, %v", ord.Account, err)
		return
	}

	if ord.StrategyId != "" {
		s, err := keeper.svcCtx.StrategyModel.FindByUserIdGUID(keeper.ctx, w.UserId, ord.StrategyId)
		if err != nil {
			logger.Errorf("[OrderKeeper] 查询策略信息失败, userId: %d, strategyId: %s, %v", w.UserId, ord.StrategyId, err)
			return
		}

		if !force && !s.EnablePushNotification {
			return
		}
	}

	if w.UserId == 0 {
		logger.Warnf("[OrderKeeper] 用户未绑定Telegram账号, 无法发送通知")
		return
	}

	_, err = utils.SendMessage(keeper.svcCtx.BotApi, w.UserId, text)
	if err != nil {
		logger.Warnf("[OrderKeeper] 发送电报通知失败, userId: %d, text: %s, %v", w.UserId, text, err)
		return
	}
}

func (keeper *OrderKeeper) handleRetryExit(ord *ent.Order) {
	// 查询策略
	record, err := keeper.svcCtx.StrategyModel.FindByGUID(keeper.ctx, ord.StrategyId)
	if err != nil {
		logger.Errorf("[OrderKeeper] 查询策略信息失败, account: %s, strategy: %s, %v", ord.Account, ord.StrategyId, err)
		return
	}

	keeper.sendNotification(ord, fmt.Sprintf("♻️ 正在尝试重新清仓 *%s* 代币失败", ord.Symbol), true)

	// 卖出代币
	orderArgs, err := strategy.SellToken(keeper.ctx, keeper.svcCtx, record, "重新清仓", &ord.InAmount, nil, true)
	if err != nil {
		logger.Errorf("[OrderKeeper] 尝试重新清仓失败, strategy: %s, token: %s, %v", ord.StrategyId, ord.Symbol, err)
		keeper.sendNotification(ord, fmt.Sprintf("❌ 尝试重新清仓 *%s* 代币失败，请手动清仓", ord.Symbol), true)
		return
	}
	orderArgs.GridBuyCost = ord.GridBuyCost

	// 保存订单记录
	err = utils.Tx(keeper.ctx, keeper.svcCtx.DbClient, func(tx *ent.Tx) error {
		_, err = model.NewOrderModel(tx.Order).Save(keeper.ctx, orderArgs)
		return err
	})
	if err != nil {
		logger.Errorf("[OrderKeeper] 保存订单记录失败, order: %+v, %v", orderArgs, err)
	}
}

func (keeper *OrderKeeper) handleCloseOrder(ord *ent.Order, tokenBalanceChanges map[common.Address]*big.Int) {
	tokenMeta, err := keeper.svcCtx.TokenMetaCache.GetTokenMeta(keeper.ctx, ord.Token)
	if err != nil {
		logger.Errorf("[OrderKeeper] 查询代币元数据失败, token: %s, %v", ord.Token, err)
		return
	}

	// 计算最终价格
	cost := decimal.Zero
	var finalPrice, outAmount decimal.Decimal
	stablecoinCA := common.HexToAddress(keeper.svcCtx.Config.Chain.StablecoinCA)

	switch ord.Type {
	case order.TypeBuy:
		change := decimal.Zero
		v, ok := tokenBalanceChanges[common.HexToAddress(ord.Token)]
		if ok {
			change = evm.ParseUnits(v, tokenMeta.Decimals)
		}

		if ok && !change.Equal(decimal.Zero) {
			finalPrice = ord.InAmount.Div(change)
		}
		outAmount = change
	case order.TypeSell:
		change := decimal.Zero
		v, ok := tokenBalanceChanges[stablecoinCA]
		if ok {
			change = evm.ParseUnits(v, keeper.svcCtx.Config.Chain.StablecoinDecimals)
		}

		if ok && !ord.InAmount.Equal(decimal.Zero) {
			finalPrice = change.Div(ord.InAmount)
		}
		outAmount = change

		if ord.GridBuyCost != nil {
			cost = *ord.GridBuyCost
		} else if ord.GridId != nil {
			g, err := keeper.svcCtx.GridModel.FindByGuid(keeper.ctx, *ord.GridId)
			if err == nil {
				cost = g.Amount
			} else {
				logger.Errorf("[OrderKeeper] 查询网格信息失败, guid: %s, %v", *ord.GridId, err)
			}
		}
	}

	// 获取策略信息
	s, err := keeper.svcCtx.StrategyModel.FindByGUID(keeper.ctx, ord.StrategyId)
	if err != nil {
		logger.Errorf("[OrderKeeper] 查询策略信息失败, guid: %s, %v", ord.StrategyId, err)
	}

	// 查询稳定币余额
	bal, err := evm.GetTokenBalance(keeper.ctx, keeper.svcCtx.EthClient, stablecoinCA.Hex(), ord.Account)
	if err != nil {
		logger.Errorf("[OrderKeeper] 查询代币余额失败, token: %s, %v", stablecoinCA, err)
		return
	}
	stablecoinBal := evm.ParseUnits(bal, keeper.svcCtx.Config.Chain.StablecoinDecimals)

	// 更新订单状态
	err = utils.Tx(keeper.ctx, keeper.svcCtx.DbClient, func(tx *ent.Tx) error {
		if ord.GridId != nil {
			switch ord.Type {
			case order.TypeBuy:
				err := model.NewGridModel(tx.Grid).SetBoughtStatus(
					keeper.ctx, *ord.GridId, finalPrice, outAmount)
				if err != nil {
					return err
				}
			case order.TypeSell:
				_, err := model.NewGridModel(tx.Grid).DeleteByGuid(keeper.ctx, *ord.GridId)
				if err != nil {
					return err
				}
			}
		}

		if !cost.IsZero() {
			profit := outAmount.Sub(cost)
			err = model.NewOrderModel(tx.Order).UpdateProfit(keeper.ctx, ord.ID, profit)
			if err != nil {
				return err
			}
		}

		err = model.NewOrderModel(tx.Order).SetOrderClosedStatus(keeper.ctx, ord.ID, finalPrice, outAmount)
		if err != nil {
			return err
		}

		if s != nil && ord.GridId != nil && s.FirstOrderId == nil {
			err = model.NewStrategyModel(tx.Strategy).UpdateFirstOrderId(keeper.ctx, s.ID, &ord.ID)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		logger.Errorf("[OrderKeeper] 设置订单 closed 状态失败, id: %d, hash: %s, %v", ord.ID, ord.TxHash, err)
		return
	}
	logger.Infof("[OrderKeeper] 设置订单 closed 状态, id: %d, type: %s, finalPrice: %s, outAmount: %s, hash: %s",
		ord.ID, ord.Type, finalPrice, outAmount, ord.TxHash)

	// 发送电报通知
	chainId := keeper.svcCtx.Config.Chain.Id
	switch ord.Type {
	case order.TypeBuy:
		usdChange := decimal.Zero
		v, ok := tokenBalanceChanges[stablecoinCA]
		if ok {
			usdChange = evm.ParseUnits(v, keeper.svcCtx.Config.Chain.StablecoinDecimals)
		}

		text := fmt.Sprintf("🟢 网格 `#%d` 买入 %sU [%s](%s) 💰 余额: %sU [>>](%s)",
			*ord.GridNumber,
			usdChange.Abs().Truncate(2),
			ord.Symbol,
			utils.GetGmgnTokenLink(chainId, ord.Token),
			stablecoinBal.Truncate(2),
			utils.GetBlockExplorerTxLink(chainId, ord.TxHash),
		)
		keeper.sendNotification(ord, text, false)
	case order.TypeSell:
		if ord.GridId != nil {
			usdChange := decimal.Zero
			v, ok := tokenBalanceChanges[stablecoinCA]
			if ok {
				usdChange = evm.ParseUnits(v, keeper.svcCtx.Config.Chain.StablecoinDecimals)
			}

			text := fmt.Sprintf("🔴 网格 `#%d` 卖出 %sU [%s](%s) 💰 余额: %sU [>>](%s)",
				*ord.GridNumber,
				usdChange.Abs().Truncate(2),
				ord.Symbol,
				utils.GetGmgnTokenLink(chainId, ord.Token),
				stablecoinBal.Truncate(2),
				utils.GetBlockExplorerTxLink(chainId, ord.TxHash),
			)
			keeper.sendNotification(ord, text, false)
		} else {
			text := fmt.Sprintf("✅ 清仓 *%s* 代币成功, 成交价格: %s, 💰 金额: %sU, 💰 余额: %sU [>>](%s))",
				ord.Symbol, format.Price(finalPrice, 5), outAmount.Truncate(2), stablecoinBal.Truncate(2), utils.GetBlockExplorerTxLink(chainId, ord.TxHash))
			keeper.sendNotification(ord, text, true)
		}
	}
}

func (keeper *OrderKeeper) handleRejectOrder(ord *ent.Order, reason string) {
	err := utils.Tx(keeper.ctx, keeper.svcCtx.DbClient, func(tx *ent.Tx) error {
		if ord.GridId != nil {
			if ord.Type == order.TypeBuy {
				_, err := model.NewGridModel(tx.Grid).DeleteByGuid(keeper.ctx, *ord.GridId)
				if err != nil {
					return err
				}
			} else {
				err := model.NewGridModel(tx.Grid).UpdateStatusByGuid(keeper.ctx, *ord.GridId, grid.StatusBought)
				if err != nil {
					return err
				}
			}
		}

		return model.NewOrderModel(tx.Order).SetOrderRejectedStatus(keeper.ctx, ord.ID, reason)
	})
	if err != nil {
		logger.Errorf("[OrderKeeper] 设置订单 rejected 状态失败, id: %d, hash: %s, %v", ord.ID, ord.TxHash, err)
		return
	}
	logger.Infof("[OrderKeeper] 设置订单 rejected 状态, id: %d, hash: %s, reason: %s", ord.ID, ord.TxHash, reason)

	// 发送失败通知
	chainId := keeper.svcCtx.Config.Chain.Id
	switch ord.Type {
	case order.TypeBuy:
		keeper.sendNotification(ord, fmt.Sprintf("❌ 网格 `#%d` 买入 %sU [%s](%s), 原因: 流动性不足或者滑点问题 [>>](%s)",
			*ord.GridNumber, ord.InAmount.Truncate(2), ord.Symbol, utils.GetGmgnTokenLink(chainId, ord.Token), utils.GetBlockExplorerTxLink(chainId, ord.TxHash)), false)
	case order.TypeSell:
		if ord.GridId != nil {
			keeper.sendNotification(ord, fmt.Sprintf("❌ 网格 `#%d` 卖出 %s [%s](%s) 失败, 原因: 流动性不足或者滑点问题 [>>](%s)",
				*ord.GridNumber, ord.InAmount, ord.Symbol, utils.GetGmgnTokenLink(chainId, ord.Token), utils.GetBlockExplorerTxLink(chainId, ord.TxHash)), false)
		} else {
			keeper.sendNotification(ord, fmt.Sprintf("❌ 清仓 *%s* 代币失败, 原因: 流动性不足或者滑点问题 [>>](%s)", ord.Symbol, utils.GetBlockExplorerTxLink(chainId, ord.TxHash)), true)
		}
	}

	// 重试清仓操作
	if ord.Type == order.TypeSell && ord.GridId == nil {
		keeper.handleRetryExit(ord)
	}
}

func (keeper *OrderKeeper) handlePolling() {
	// 获取订单列表
	orders, err := keeper.svcCtx.OrderModel.FindPendingOrders(keeper.ctx, 100)
	if err != nil {
		logger.Errorf("[OrderKeeper] 获取订单列表失败, %v", err)
	}
	if len(orders) == 0 {
		return
	}

	// 检查交易状态
	now := time.Now()
	openOrders := make([]*ent.Order, 0)
	tokenBalanceChanges := make(map[int]map[common.Address]*big.Int)
	for _, item := range orders {
		// 忽略超时交易
		_, timeout := keeper.timeoutTxs[item.TxHash]
		if timeout {
			continue
		}

		// 查询交易收据
		receipt, err := keeper.svcCtx.EthClient.TransactionReceipt(keeper.ctx, common.HexToHash(item.TxHash))
		if err != nil {
			// 标记超时交易
			if strings.Contains(err.Error(), "not found") {
				if now.Sub(item.CreateTime) > time.Minute*2 {
					keeper.timeoutTxs[item.TxHash] = struct{}{}
					logger.Errorf("[OrderKeeper] 交易打包超时, account: %s, nonce: %d, hash: %s, createTime: %v",
						item.Account, item.Nonce, item.TxHash, item.CreateTime)
				}
				continue
			}

			logger.Errorf("[OrderKeeper] 查询交易收据失败, account: %s, nonce: %d, hash: %s, %v", item.Account, item.Nonce, item.TxHash, err)
			return
		}

		// 处理驳回订单
		if receipt.Status == 0 {
			keeper.handleRejectOrder(item, "execution reverted")
			continue
		}

		// 查询余额变化
		changes, err := evm.GetTokenBalanceChanges(keeper.ctx, keeper.svcCtx.EthClient, receipt, item.Account)
		if err != nil {
			logger.Errorf("[OrderKeeper] 查询代币余额变化失败, account: %s, nonce: %d, hash: %s, %v", item.Account, item.Nonce, item.TxHash, err)
			return
		}

		openOrders = append(openOrders, item)
		tokenBalanceChanges[item.ID] = changes
	}

	if len(openOrders) == 0 {
		return
	}

	// 更新订单信息
	for _, item := range openOrders {
		changes, ok := tokenBalanceChanges[item.ID]
		if !ok {
			continue
		}
		keeper.handleCloseOrder(item, changes)
	}
}
