package model

import (
	"context"
	"time"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/strategy"

	"entgo.io/ent/dialect/sql"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type StrategyModel struct {
	client *ent.StrategyClient
}

func NewStrategyModel(client *ent.StrategyClient) *StrategyModel {
	return &StrategyModel{client: client}
}

func (model *StrategyModel) Save(ctx context.Context, args ent.Strategy) (*ent.Strategy, error) {
	return model.client.Create().
		SetGUID(args.GUID).
		SetUserId(args.UserId).
		SetToken(common.HexToAddress(args.Token).Hex()).
		SetSymbol(args.Symbol).
		SetMartinFactor(args.MartinFactor).
		SetNillableMaxGridLimit(args.MaxGridLimit).
		SetTakeProfitRatio(args.TakeProfitRatio).
		SetLowerPriceBound(args.LowerPriceBound).
		SetUpperPriceBound(args.UpperPriceBound).
		SetInitialOrderSize(args.InitialOrderSize).
		SetNillableFirstOrderId(args.FirstOrderId).
		SetNillableUpperBoundExit(args.UpperBoundExit).
		SetNillableStopLossExit(args.StopLossExit).
		SetNillableTakeProfitExit(args.TakeProfitExit).
		SetNillableLastKlineVolume(args.LastKlineVolume).
		SetNillableFiveKlineVolume(args.FiveKlineVolume).
		SetNillableGlobalTakeProfitRatio(args.GlobalTakeProfitRatio).
		SetDropOn(args.DropOn).
		SetCandlesToCheck(args.CandlesToCheck).
		SetNillableDropThreshold(args.DropThreshold).
		SetEnableAutoBuy(args.EnableAutoBuy).
		SetEnableAutoSell(args.EnableAutoSell).
		SetEnableAutoExit(args.EnableAutoExit).
		SetEnablePushNotification(args.EnablePushNotification).
		SetStatus(args.Status).
		SetNillableGridTrend(args.GridTrend).
		SetNillableLastLowerThresholdAlertTime(args.LastLowerThresholdAlertTime).
		SetNillableLastUpperThresholdAlertTime(args.LastUpperThresholdAlertTime).
		Save(ctx)
}

func (model *StrategyModel) FindByGUID(ctx context.Context, guid string) (*ent.Strategy, error) {
	return model.client.Query().
		Where(strategy.GUIDEQ(guid)).
		First(ctx)
}

func (model *StrategyModel) FindAllActive(ctx context.Context, offset, limit int) ([]*ent.Strategy, error) {
	return model.client.Query().
		Where(strategy.StatusEQ(strategy.StatusActive)).
		Order(strategy.ByID(sql.OrderAsc())).
		Offset(offset).
		Limit(limit).
		All(ctx)
}

func (model *StrategyModel) FindByUserIdGUID(ctx context.Context, userId int64, guid string) (*ent.Strategy, error) {
	return model.client.Query().
		Where(strategy.UserIdEQ(userId), strategy.GUIDEQ(guid)).
		First(ctx)
}

func (model *StrategyModel) FindByUserIdToken(ctx context.Context, userId int64, token string) (*ent.Strategy, error) {
	return model.client.Query().
		Where(strategy.UserIdEQ(userId), strategy.TokenEQ(common.HexToAddress(token).Hex())).
		First(ctx)
}

func (model *StrategyModel) FindByUserId(ctx context.Context, userId int64, offset, limit int) ([]*ent.Strategy, int, error) {
	query := model.client.Query().
		Where(strategy.UserIdEQ(userId))
	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	data, err := query.Order(strategy.ByID(sql.OrderDesc())).Offset(offset).Limit(limit).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return data, count, nil
}

func (model *StrategyModel) UpdateEnableAutoBuy(ctx context.Context, id int, newValue bool) error {
	return model.client.UpdateOneID(id).SetEnableAutoBuy(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateEnableAutoSell(ctx context.Context, id int, newValue bool) error {
	return model.client.UpdateOneID(id).SetEnableAutoSell(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateEnableAutoExit(ctx context.Context, id int, newValue bool) error {
	return model.client.UpdateOneID(id).SetEnableAutoExit(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateDynamicStopLoss(ctx context.Context, id int, newValue bool) error {
	return model.client.UpdateOneID(id).SetDynamicStopLoss(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateGridTrend(ctx context.Context, id int, trending string) error {
	return model.client.UpdateOneID(id).SetGridTrend(trending).Exec(ctx)
}

func (model *StrategyModel) UpdateEnablePushNotification(ctx context.Context, id int, newValue bool) error {
	return model.client.UpdateOneID(id).SetEnablePushNotification(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateStatusByGuid(ctx context.Context, guid string, newValue strategy.Status) error {
	return model.client.Update().Where(strategy.GUIDEQ(guid)).SetStatus(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateInitialOrderSize(ctx context.Context, id int, newValue decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetInitialOrderSize(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateMaxGridLimit(ctx context.Context, id int, newValue int) error {
	return model.client.UpdateOneID(id).SetMaxGridLimit(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateTakeProfitRatio(ctx context.Context, id int, newValue decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetTakeProfitRatio(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateFirstOrderId(ctx context.Context, id int, newValue *int) error {
	if newValue == nil {
		return model.client.UpdateOneID(id).ClearFirstOrderId().Exec(ctx)
	}
	return model.client.UpdateOneID(id).SetFirstOrderId(*newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateDropOn(ctx context.Context, id int, newValue bool) error {
	return model.client.UpdateOneID(id).SetDropOn(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateCandlesToCheck(ctx context.Context, id int, candlesToCheck int) error {
	return model.client.UpdateOneID(id).SetCandlesToCheck(candlesToCheck).Exec(ctx)
}

func (model *StrategyModel) UpdateDropThreshold(ctx context.Context, id int, dropThreshold decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetDropThreshold(dropThreshold).Exec(ctx)
}

func (model *StrategyModel) UpdateUpperBoundExit(ctx context.Context, id int, newValue decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetUpperBoundExit(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateStopLossExit(ctx context.Context, id int, newValue decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetStopLossExit(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateTakeProfitExit(ctx context.Context, id int, newValue decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetTakeProfitExit(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateUpperPriceBound(ctx context.Context, id int, newValue decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetUpperPriceBound(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateLowerPriceBound(ctx context.Context, id int, newValue decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetLowerPriceBound(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateLastKlineVolume(ctx context.Context, id int, newValue decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetLastKlineVolume(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateFiveKlineVolume(ctx context.Context, id int, newValue decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetFiveKlineVolume(newValue).Exec(ctx)
}

func (model *StrategyModel) ClearLastLowerThresholdAlertTime(ctx context.Context, id int) error {
	return model.client.UpdateOneID(id).ClearLastLowerThresholdAlertTime().Exec(ctx)
}

func (model *StrategyModel) UpdateLastLowerThresholdAlertTime(ctx context.Context, id int, newValue time.Time) error {
	return model.client.UpdateOneID(id).SetLastLowerThresholdAlertTime(newValue).Exec(ctx)
}

func (model *StrategyModel) ClearLastUpperThresholdAlertTime(ctx context.Context, id int) error {
	return model.client.UpdateOneID(id).ClearLastUpperThresholdAlertTime().Exec(ctx)
}

func (model *StrategyModel) UpdateLastUpperThresholdAlertTime(ctx context.Context, id int, newValue time.Time) error {
	return model.client.UpdateOneID(id).SetLastUpperThresholdAlertTime(newValue).Exec(ctx)
}

func (model *StrategyModel) UpdateGlobalTakeProfitRatio(ctx context.Context, id int, newValue decimal.Decimal) error {
	return model.client.UpdateOneID(id).SetGlobalTakeProfitRatio(newValue).Exec(ctx)
}

func (model *StrategyModel) Delete(ctx context.Context, id int) error {
	return model.client.DeleteOneID(id).Exec(ctx)
}
