package model

import (
	"context"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/wallet"

	"github.com/ethereum/go-ethereum/common"
)

type WalletModel struct {
	client *ent.WalletClient
}

func NewWalletModel(client *ent.WalletClient) *WalletModel {
	return &WalletModel{client: client}
}

func (model *WalletModel) Save(ctx context.Context, args ent.Wallet) (*ent.Wallet, error) {
	return model.client.Create().
		SetUserId(args.UserId).
		SetAccount(common.HexToAddress(args.Account).Hex()).
		SetPassword(args.Password).
		SetPrivateKey(args.PrivateKey).
		Save(ctx)
}

func (model *WalletModel) FindByUserId(ctx context.Context, userId int64) (*ent.Wallet, error) {
	return model.client.Query().
		Where(wallet.UserIdEQ(userId)).
		First(ctx)
}

func (model *WalletModel) FindByAccount(ctx context.Context, account string) (*ent.Wallet, error) {
	return model.client.Query().
		Where(wallet.AccountEQ(common.HexToAddress(account).Hex())).
		First(ctx)
}

func (model *WalletModel) UpdatePassword(ctx context.Context, account, password string) error {
	return model.client.Update().
		Where(wallet.AccountEQ(common.HexToAddress(account).Hex())).
		SetPassword(password).
		Exec(ctx)
}
