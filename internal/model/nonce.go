package model

import (
	"context"

	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/ent/nonce"

	"github.com/ethereum/go-ethereum/common"
)

type NonceModel struct {
	client *ent.NonceClient
}

func NewNonceModel(client *ent.NonceClient) *NonceModel {
	return &NonceModel{client: client}
}

func (m *NonceModel) Save(ctx context.Context, account string, n uint64) error {
	return m.client.Create().
		SetAccount(common.HexToAddress(account).Hex()).
		SetNonce(n).
		Exec(ctx)
}

func (m *NonceModel) FindOne(ctx context.Context, account string) (*ent.Nonce, error) {
	return m.client.Query().Where(nonce.AccountEQ(common.HexToAddress(account).Hex())).First(ctx)
}

func (m *NonceModel) UpdateNonce(ctx context.Context, account string, newValue uint64) error {
	return m.client.Update().SetNonce(newValue).Where(nonce.AccountEQ(common.HexToAddress(account).Hex())).Exec(ctx)
}
