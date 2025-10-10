package model

import (
	"context"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/settings"
)

type SettingsModel struct {
	client *ent.SettingsClient
}

func NewSettingsModel(client *ent.SettingsClient) *SettingsModel {
	return &SettingsModel{client: client}
}

func (model *SettingsModel) Save(ctx context.Context, args ent.Settings) (*ent.Settings, error) {
	return model.client.Create().
		SetUserId(args.UserId).
		SetSlippageBps(args.SlippageBps).
		SetNillableSellSlippageBps(args.SellSlippageBps).
		SetNillableExitSlippageBps(args.ExitSlippageBps).
		SetDexAggregator(args.DexAggregator).
		SetNillableEnableInfiniteApproval(args.EnableInfiniteApproval).
		Save(ctx)
}

func (model *SettingsModel) FindByUserId(ctx context.Context, userId int64) (*ent.Settings, error) {
	return model.client.Query().
		Where(settings.UserIdEQ(userId)).
		First(ctx)
}

func (model *SettingsModel) UpdateSlippageBps(ctx context.Context, id int, newValue int) error {
	return model.client.UpdateOneID(id).
		SetSlippageBps(newValue).
		Exec(ctx)
}

func (model *SettingsModel) UpdateSellSlippageBps(ctx context.Context, id int, newValue int) error {
	return model.client.UpdateOneID(id).
		SetSellSlippageBps(newValue).
		Exec(ctx)
}

func (model *SettingsModel) UpdateExitSlippageBps(ctx context.Context, id int, newValue int) error {
	return model.client.UpdateOneID(id).
		SetExitSlippageBps(newValue).
		Exec(ctx)
}

func (model *SettingsModel) UpdateDexAggregator(ctx context.Context, id int, dexAggregator settings.DexAggregator) error {
	return model.client.UpdateOneID(id).
		SetDexAggregator(dexAggregator).
		Exec(ctx)
}

func (model *SettingsModel) UpdateEnableInfiniteApproval(ctx context.Context, id int, newValue bool) error {
	return model.client.UpdateOneID(id).
		SetEnableInfiniteApproval(newValue).
		Exec(ctx)
}
