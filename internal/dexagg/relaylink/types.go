package relaylink

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

type Currency struct {
	ID               string `json:"id"`
	Symbol           string `json:"symbol"`
	Name             string `json:"name"`
	Address          string `json:"address"`
	Decimals         int    `json:"decimals"`
	SupportsBridging bool   `json:"supportsBridging"`
}

type FeaturedToken struct {
	ID               string `json:"id"`
	Symbol           string `json:"symbol"`
	Name             string `json:"name"`
	Address          string `json:"address"`
	Decimals         int    `json:"decimals"`
	SupportsBridging bool   `json:"supportsBridging"`
	Metadata         struct {
		LogoURI string `json:"logoURI"`
	} `json:"metadata"`
}

type ERC20Currency struct {
	ID               string `json:"id"`
	Symbol           string `json:"symbol"`
	Name             string `json:"name"`
	Address          string `json:"address"`
	Decimals         int    `json:"decimals"`
	SupportsBridging bool   `json:"supportsBridging"`
	SupportsPermit   bool   `json:"supportsPermit"`
	WithdrawalFee    int    `json:"withdrawalFee"`
	DepositFee       int    `json:"depositFee"`
	SurgeEnabled     bool   `json:"surgeEnabled"`
}

type Contracts struct {
	Multicall3           string `json:"multicall3"`
	Multicaller          string `json:"multicaller"`
	OnlyOwnerMulticaller string `json:"onlyOwnerMulticaller"`
	RelayReceiver        string `json:"relayReceiver"`
	Erc20Router          string `json:"erc20Router"`
	ApprovalProxy        string `json:"approvalProxy"`
}

type ExplorerPaths struct {
	Transaction string `json:"transaction"`
}

type Chain struct {
	ID                     int64             `json:"id"`
	Name                   string            `json:"name"`
	DisplayName            string            `json:"displayName"`
	HTTPRpcUrl             string            `json:"httpRpcUrl"`
	WSRpcUrl               string            `json:"wsRpcUrl"`
	ExplorerUrl            string            `json:"explorerUrl"`
	ExplorerName           string            `json:"explorerName"`
	ExplorerPaths          ExplorerPaths     `json:"explorerPaths"`
	DepositEnabled         bool              `json:"depositEnabled"`
	TokenSupport           string            `json:"tokenSupport"`
	Disabled               bool              `json:"disabled"`
	PartialDisableLimit    int               `json:"partialDisableLimit"`
	BlockProductionLagging bool              `json:"blockProductionLagging"`
	Currency               Currency          `json:"currency"`
	WithdrawalFee          int               `json:"withdrawalFee"`
	DepositFee             int               `json:"depositFee"`
	SurgeEnabled           bool              `json:"surgeEnabled"`
	FeaturedTokens         []FeaturedToken   `json:"featuredTokens"`
	ERC20Currencies        []ERC20Currency   `json:"erc20Currencies"`
	IconUrl                string            `json:"iconUrl"`
	LogoUrl                string            `json:"logoUrl"`
	BrandColor             string            `json:"brandColor"`
	Contracts              Contracts         `json:"contracts"`
	VMType                 string            `json:"vmType"`
	ExplorerQueryParams    map[string]string `json:"explorerQueryParams"`
	BaseChainId            int               `json:"baseChainId"`
	StatusMessage          string            `json:"statusMessage"`
	Tags                   []string          `json:"tags"`
}

type Chains struct {
	Chains []Chain `json:"chains"`
}

type CurrencyMetadata struct {
	LogoURI  string `json:"logoURI"`
	Verified bool   `json:"verified"`
	IsNative bool   `json:"isNative"`
}

type FeeCurrency struct {
	ChainID  int              `json:"chainId"`
	Address  string           `json:"address"`
	Symbol   string           `json:"symbol"`
	Name     string           `json:"name"`
	Decimals int              `json:"decimals"`
	Metadata CurrencyMetadata `json:"metadata"`
}

type FeeDetails struct {
	Currency        FeeCurrency     `json:"currency"`
	Amount          decimal.Decimal `json:"amount"`
	AmountFormatted decimal.Decimal `json:"amountFormatted"`
	AmountUsd       decimal.Decimal `json:"amountUsd"`
	MinimumAmount   decimal.Decimal `json:"minimumAmount"`
}

type SolKey struct {
	Pubkey     string `json:"pubkey"`
	IsSigner   bool   `json:"isSigner"`
	IsWritable bool   `json:"isWritable"`
}

type SolInstruction struct {
	ProgramId string   `json:"programId"`
	Keys      []SolKey `json:"keys"`
	Data      string   `json:"data"`
}

type TransactionData struct {
	// evm
	EvmFrom                 common.Address  `json:"from"`
	EvmTo                   common.Address  `json:"to"`
	EvmGas                  decimal.Decimal `json:"gas"`
	EvmData                 string          `json:"data"`
	EvmValue                decimal.Decimal `json:"value"`
	EvmMaxFeePerGas         decimal.Decimal `json:"maxFeePerGas"`
	EvmMaxPriorityFeePerGas decimal.Decimal `json:"maxPriorityFeePerGas"`
	EvmChainID              int             `json:"chainId"`

	// sol
	SolInstructions                []SolInstruction `json:"instructions"`
	SolAddressLookupTableAddresses []string         `json:"addressLookupTableAddresses"`
}

type Check struct {
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
}

type Item struct {
	Status string          `json:"status"`
	Data   TransactionData `json:"data"`
	Check  Check           `json:"check"`
}

type Step struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
	RequestID   string `json:"requestId"`
	Items       []Item `json:"items"`
}

type TotalImpact struct {
	Usd     string `json:"usd"`
	Percent string `json:"percent"`
}

type SlippageTolerance struct {
	Origin struct {
		Usd     decimal.Decimal `json:"usd"`
		Value   decimal.Decimal `json:"value"`
		Percent decimal.Decimal `json:"percent"`
	} `json:"origin"`
	Destination struct {
		Usd     decimal.Decimal `json:"usd"`
		Value   decimal.Decimal `json:"value"`
		Percent decimal.Decimal `json:"percent"`
	} `json:"destination"`
}

type Details struct {
	Operation         string            `json:"operation"`
	Sender            string            `json:"sender"`
	Recipient         string            `json:"recipient"`
	CurrencyIn        FeeDetails        `json:"currencyIn"`
	CurrencyOut       FeeDetails        `json:"currencyOut"`
	CurrencyGasTopup  FeeDetails        `json:"currencyGasTopup"`
	TotalImpact       TotalImpact       `json:"totalImpact"`
	SwapImpact        TotalImpact       `json:"swapImpact"`
	Rate              string            `json:"rate"`
	SlippageTolerance SlippageTolerance `json:"slippageTolerance"`
	TimeEstimate      int               `json:"timeEstimate"`
	UserBalance       string            `json:"userBalance"`
}

type Fees struct {
	Gas            FeeDetails `json:"gas"`
	Relayer        FeeDetails `json:"relayer"`
	RelayerGas     FeeDetails `json:"relayerGas"`
	RelayerService FeeDetails `json:"relayerService"`
	App            FeeDetails `json:"app"`
}

type QuoteResponse struct {
	Steps   []Step  `json:"steps"`
	Fees    Fees    `json:"fees"`
	Details Details `json:"details"`
}

type ErrorResponse struct {
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("message: %s, errorCode: %s", e.Message, e.ErrorCode)
}
