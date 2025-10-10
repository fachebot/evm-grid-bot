package cache

import (
	"context"
	"sync"

	"github.com/fachebot/evm-grid-bot/internal/utils/evm"

	"github.com/ethereum/go-ethereum/ethclient"
)

type TokenMeta struct {
	Name     string
	Symbol   string
	Decimals uint8
}

type TokenMetaCache struct {
	client       *ethclient.Client
	tokenMetaMap sync.Map
}

func NewTokenMetaCache(client *ethclient.Client) *TokenMetaCache {
	return &TokenMetaCache{client: client}
}

func (c *TokenMetaCache) GetTokenMeta(ctx context.Context, tokenAddress string) (TokenMeta, error) {
	meta, err := c.loadTokenMeta(ctx, tokenAddress)
	if err != nil {
		return TokenMeta{}, err
	}
	return meta, nil
}

func (c *TokenMetaCache) loadTokenMeta(ctx context.Context, tokenAddress string) (TokenMeta, error) {
	val, ok := c.tokenMetaMap.Load(tokenAddress)
	if ok {
		return val.(TokenMeta), nil
	}

	tokenmeta, err := evm.GetTokenMeta(ctx, c.client, tokenAddress)
	if err != nil {
		return TokenMeta{}, err
	}

	ret := TokenMeta{
		Name:     tokenmeta.Name,
		Symbol:   tokenmeta.Symbol,
		Decimals: tokenmeta.Decimals,
	}
	c.tokenMetaMap.Store(tokenAddress, ret)

	return ret, nil
}
