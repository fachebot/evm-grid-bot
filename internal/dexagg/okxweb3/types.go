package okxweb3

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

type restResponse struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data,omitempty"`
}

type ChainInfo struct {
	Name       string `json:"name"`
	LogoUrl    string `json:"logoUrl"`
	ShortName  string `json:"shortName"`
	ChainIndex string `json:"chainIndex"`
}

type RealtimePrice struct {
	ChainIndex   string          `json:"chainIndex"`
	TokenAddress string          `json:"tokenAddress"`
	Time         string          `json:"time"`
	Price        decimal.Decimal `json:"price"`
}

type TokenBalance struct {
	ChainIndex           string          `json:"chainIndex"`
	TokenContractAddress string          `json:"tokenContractAddress"`
	Symbol               string          `json:"symbol"`
	Balance              decimal.Decimal `json:"balance"`
	TokenPrice           decimal.Decimal `json:"tokenPrice"`
	IsRiskToken          bool            `json:"isRiskToken"`
	RawBalance           string          `json:"rawBalance"`
	Address              string          `json:"address"`
}

type TokenAssets struct {
	TokenAssets []TokenBalance `json:"tokenAssets"`
}

type Account struct {
	IsSigner   bool   `json:"isSigner"`
	IsWritable bool   `json:"isWritable"`
	Pubkey     string `json:"pubkey"`
}

type Instruction struct {
	Data      string    `json:"data"`
	Accounts  []Account `json:"accounts"`
	ProgramId string    `json:"programId"`
}

type DexProtocol struct {
	DexName string `json:"dexName"`
	Percent string `json:"percent"`
}

type Token struct {
	Decimal              decimal.Decimal `json:"decimal"`
	IsHoneyPot           bool            `json:"isHoneyPot"`
	TaxRate              decimal.Decimal `json:"taxRate"`
	TokenContractAddress string          `json:"tokenContractAddress"`
	TokenSymbol          string          `json:"tokenSymbol"`
	TokenUnitPrice       decimal.Decimal `json:"tokenUnitPrice"`
}

type SubRouter struct {
	DexProtocol []DexProtocol `json:"dexProtocol"`
	FromToken   Token         `json:"fromToken"`
	ToToken     Token         `json:"toToken"`
}

type DexRouter struct {
	Router        string      `json:"router"`
	RouterPercent string      `json:"routerPercent"`
	SubRouterList []SubRouter `json:"subRouterList"`
}

type Quote struct {
	AmountOut string `json:"amountOut"`
	DexLogo   string `json:"dexLogo"`
	DexName   string `json:"dexName"`
	TradeFee  string `json:"tradeFee"`
}

type RouterResult struct {
	ChainId               string          `json:"chainId"`
	ChainIndex            string          `json:"chainIndex"`
	DexRouterList         []DexRouter     `json:"dexRouterList"`
	EstimateGasFee        decimal.Decimal `json:"estimateGasFee"`
	FromToken             Token           `json:"fromToken"`
	FromTokenAmount       decimal.Decimal `json:"fromTokenAmount"`
	PriceImpactPercentage decimal.Decimal `json:"priceImpactPercentage"`
	QuoteCompareList      []Quote         `json:"quoteCompareList"`
	SwapMode              string          `json:"swapMode"`
	ToToken               Token           `json:"toToken"`
	ToTokenAmount         decimal.Decimal `json:"toTokenAmount"`
	TradeFee              decimal.Decimal `json:"tradeFee"`
}

type Transaction struct {
	From             string          `json:"from"`
	MinReceiveAmount decimal.Decimal `json:"minReceiveAmount"`
	Slippage         decimal.Decimal `json:"slippage"`
	To               string          `json:"to"`
}

type SwapInstruction struct {
	AddressLookupTableAccount []string      `json:"addressLookupTableAccount"`
	InstructionLists          []Instruction `json:"instructionLists"`
	RouterResult              RouterResult  `json:"routerResult"`
	Tx                        Transaction   `json:"tx"`
}
