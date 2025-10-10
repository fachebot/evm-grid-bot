package model

import (
	"context"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/order"

	"entgo.io/ent/dialect/sql"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type OrderModel struct {
	client *ent.OrderClient
}

func NewOrderModel(client *ent.OrderClient) *OrderModel {
	return &OrderModel{client: client}
}

func (model *OrderModel) Save(ctx context.Context, args ent.Order) (*ent.Order, error) {
	return model.client.Create().
		SetAccount(common.HexToAddress(args.Account).Hex()).
		SetToken(common.HexToAddress(args.Token).Hex()).
		SetSymbol(args.Symbol).
		SetNillableGridId(args.GridId).
		SetNillableGridNumber(args.GridNumber).
		SetNillableGridBuyCost(args.GridBuyCost).
		SetStrategyId(args.StrategyId).
		SetType(args.Type).
		SetPrice(args.Price).
		SetFinalPrice(args.FinalPrice).
		SetInAmount(args.InAmount).
		SetOutAmount(args.OutAmount).
		SetStatus(args.Status).
		SetNonce(args.Nonce).
		SetTxHash(args.TxHash).
		SetReason(args.Reason).
		SetNillableProfit(args.Profit).
		Save(ctx)
}

func (model *OrderModel) TotalProfit(ctx context.Context, strategyId string, firstOrderId int) (decimal.Decimal, error) {
	orders, err := model.client.Query().
		Where(order.StrategyIdEQ(strategyId), order.IDGTE(firstOrderId)).
		All(ctx)
	if err != nil {
		return decimal.Zero, err
	}

	var totalProfit decimal.Decimal
	for _, ord := range orders {
		if ord.Profit == nil {
			continue
		}
		totalProfit = totalProfit.Add(*ord.Profit)
	}
	return totalProfit, nil
}

func (model *OrderModel) FindPendingOrders(ctx context.Context, limit int) ([]*ent.Order, error) {
	return model.client.Query().
		Where(order.StatusEQ(order.StatusPending)).
		Order(order.ByID(sql.OrderAsc())).
		Limit(limit).
		All(ctx)
}

func (model *OrderModel) FindOrdersByStrategyId(ctx context.Context, strategyId string, offset, limit int) ([]*ent.Order, int, error) {
	q := model.client.Query().
		Where(order.StrategyIdEQ(strategyId))
	count, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	data, err := q.Order(order.ByID(sql.OrderDesc())).Offset(offset).Limit(limit).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return data, count, nil
}

func (model *OrderModel) UpdateProfit(ctx context.Context, id int, profit decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetProfit(profit).Exec(ctx)
}

func (model *OrderModel) SetOrderRejectedStatus(ctx context.Context, id int, reason string) error {
	return model.client.UpdateOneID(id).SetStatus(order.StatusRejected).SetReason(reason).Exec(ctx)
}

func (model *OrderModel) SetOrderClosedStatus(ctx context.Context, id int, finalPrice, outAmount decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetStatus(order.StatusClosed).SetFinalPrice(finalPrice).SetOutAmount(outAmount).Exec(ctx)
}
