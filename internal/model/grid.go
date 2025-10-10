package model

import (
	"context"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/grid"

	"entgo.io/ent/dialect/sql"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type GridModel struct {
	client *ent.GridClient
}

func NewGridModel(client *ent.GridClient) *GridModel {
	return &GridModel{client: client}
}

func (model *GridModel) Save(ctx context.Context, args ent.Grid) (*ent.Grid, error) {
	return model.client.Create().
		SetGUID(args.GUID).
		SetAccount(common.HexToAddress(args.Account).Hex()).
		SetToken(common.HexToAddress(args.Token).Hex()).
		SetSymbol(args.Symbol).
		SetStrategyId(args.StrategyId).
		SetGridNumber(args.GridNumber).
		SetOrderPrice(args.OrderPrice).
		SetFinalPrice(args.FinalPrice).
		SetAmount(args.Amount).
		SetQuantity(args.Quantity).
		SetStatus(args.Status).
		Save(ctx)
}

func (model *GridModel) FindByGuid(ctx context.Context, guid string) (*ent.Grid, error) {
	return model.client.Query().
		Where(grid.GUIDEQ(guid)).
		First(ctx)
}

func (model *GridModel) FindByStrategyId(ctx context.Context, strategyId string) ([]*ent.Grid, error) {
	return model.client.Query().
		Where(grid.StrategyIdEQ(strategyId)).
		Order(grid.ByID(sql.OrderAsc())).
		All(ctx)
}

func (model *GridModel) SetSellingStatus(ctx context.Context, guid string) error {
	return model.client.Update().
		Where(grid.GUIDEQ(guid)).
		SetStatus(grid.StatusSelling).
		Exec(ctx)
}

func (model *GridModel) SetBoughtStatus(ctx context.Context, guid string, finalPrice, quantity decimal.Decimal) error {
	return model.client.Update().
		Where(grid.GUIDEQ(guid)).
		SetFinalPrice(finalPrice).
		SetQuantity(quantity).
		SetStatus(grid.StatusBought).
		Exec(ctx)
}

func (model *GridModel) UpdateStatusByGuid(ctx context.Context, guid string, status grid.Status) error {
	return model.client.Update().
		Where(grid.GUIDEQ(guid)).
		SetStatus(status).
		Exec(ctx)
}

func (model *GridModel) DeleteByGuid(ctx context.Context, guid string) (int, error) {
	return model.client.Delete().Where(grid.GUIDEQ(guid)).Exec(ctx)
}

func (model *GridModel) DeleteByStrategyId(ctx context.Context, strategyId string) (int, error) {
	return model.client.Delete().Where(grid.StrategyId(strategyId)).Exec(ctx)
}
