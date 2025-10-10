package swap

import (
	"context"
	"math/big"

	"github.com/fachebot/evm-grid-bot/internal/dexagg/relaylink"
)

type SwapTransaction interface {
	Signer() string
	OutAmount() *big.Int
	SlippageBps() int
	Swap(ctx context.Context) (string, uint64, error)
}

type RelaySwapTransaction struct {
	quote   *relaylink.QuoteResponse
	service *SwapService
	signer  string
}

func NewRelaySwapTransaction(service *SwapService, quoteResponse *relaylink.QuoteResponse, signer string) *RelaySwapTransaction {
	return &RelaySwapTransaction{
		quote:   quoteResponse,
		service: service,
		signer:  signer,
	}
}

func (tx *RelaySwapTransaction) Signer() string {
	return tx.signer
}

func (tx *RelaySwapTransaction) OutAmount() *big.Int {
	return tx.quote.Details.CurrencyOut.Amount.BigInt()
}

func (tx *RelaySwapTransaction) SlippageBps() int {
	return int(tx.quote.Details.SlippageTolerance.Origin.Percent.RoundUp(0).IntPart())
}

func (tx *RelaySwapTransaction) Swap(ctx context.Context) (string, uint64, error) {
	userWallet, err := tx.service.getUserWallet(ctx)
	if err != nil {
		return "", 0, err
	}

	relayClient := relaylink.NewRelaylinkClient(tx.service.svcCtx.TransportProxy)
	hash, nonce, err := relayClient.SendSwapTransaction(ctx, tx.service.svcCtx, userWallet, tx.quote)
	return hash, nonce, err
}
