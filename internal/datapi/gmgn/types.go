package gmgn

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

type TokenTransfer struct {
	Name      *string `json:"name"`
	Address   string  `json:"address"`
	Timestamp int64   `json:"timestamp"`
	TxHash    string  `json:"tx_hash"`
	Type      string  `json:"type"`
}

type NativeTransfer struct {
	Name        *string `json:"name"`
	FromAddress string  `json:"from_address"`
	Timestamp   int64   `json:"timestamp"`
}

type HolderInfo struct {
	Address              string           `json:"address"`
	AccountAddress       string           `json:"account_address"`
	AddrType             int              `json:"addr_type"`
	AmountCur            decimal.Decimal  `json:"amount_cur"`
	UsdValue             decimal.Decimal  `json:"usd_value"`
	CostCur              decimal.Decimal  `json:"cost_cur"`
	SellAmountCur        decimal.Decimal  `json:"sell_amount_cur"`
	SellAmountPercentage decimal.Decimal  `json:"sell_amount_percentage"`
	SellVolumeCur        decimal.Decimal  `json:"sell_volume_cur"`
	BuyVolumeCur         decimal.Decimal  `json:"buy_volume_cur"`
	BuyAmountCur         decimal.Decimal  `json:"buy_amount_cur"`
	NetflowUsd           decimal.Decimal  `json:"netflow_usd"`
	NetflowAmount        decimal.Decimal  `json:"netflow_amount"`
	BuyTxCountCur        int              `json:"buy_tx_count_cur"`
	SellTxCountCur       int              `json:"sell_tx_count_cur"`
	WalletTagV2          string           `json:"wallet_tag_v2"`
	NativeBalance        string           `json:"native_balance"`
	Balance              decimal.Decimal  `json:"balance"`
	Profit               decimal.Decimal  `json:"profit"`
	RealizedProfit       decimal.Decimal  `json:"realized_profit"`
	ProfitChange         *decimal.Decimal `json:"profit_change"`
	AmountPercentage     decimal.Decimal  `json:"amount_percentage"`
	UnrealizedProfit     decimal.Decimal  `json:"unrealized_profit"`
	UnrealizedPnl        *decimal.Decimal `json:"unrealized_pnl"`
	AvgCost              *decimal.Decimal `json:"avg_cost"`
	AvgSold              *decimal.Decimal `json:"avg_sold"`
	AccuAmount           decimal.Decimal  `json:"accu_amount"`
	AccuCost             decimal.Decimal  `json:"accu_cost"`
	Cost                 decimal.Decimal  `json:"cost"`
	TotalCost            decimal.Decimal  `json:"total_cost"`
	TransferIn           bool             `json:"transfer_in"`
	IsNew                bool             `json:"is_new"`
	IsSuspicious         bool             `json:"is_suspicious"`
	IsOnCurve            bool             `json:"is_on_curve"`
	StartHoldingAt       int64            `json:"start_holding_at"`
	EndHoldingAt         *int64           `json:"end_holding_at"`
	LastActiveTimestamp  int64            `json:"last_active_timestamp"`
	NativeTransfer       NativeTransfer   `json:"native_transfer"`
	TokenTransfer        TokenTransfer    `json:"token_transfer"`
	TokenTransferIn      TokenTransfer    `json:"token_transfer_in"`
	TokenTransferOut     TokenTransfer    `json:"token_transfer_out"`
	Tags                 []string         `json:"tags"`
	MakerTokenTags       []string         `json:"maker_token_tags"`
	Name                 string           `json:"name"`
	Avatar               string           `json:"avatar"`
	TwitterUsername      *string          `json:"twitter_username"`
	TwitterName          *string          `json:"twitter_name"`
	CreatedAt            int64            `json:"created_at"`
}

type HolderInfoList struct {
	List []*HolderInfo `json:"list"`
}

type TokenHolding struct {
	Address           string          `json:"address"`
	TokenAddress      string          `json:"token_address"`
	Symbol            string          `json:"symbol"`
	Name              string          `json:"name"`
	Decimals          int             `json:"decimals"`
	Logo              string          `json:"logo"`
	PriceChange6h     decimal.Decimal `json:"price_change_6h"`
	IsShowAlert       bool            `json:"is_show_alert"`
	IsHoneypot        *bool           `json:"is_honeypot"`
	CreationTimestamp int64           `json:"creation_timestamp"`
	OpenTimestamp     int64           `json:"open_timestamp"`
}

type WalletHolding struct {
	Token               TokenHolding    `json:"token"`
	Balance             decimal.Decimal `json:"balance"`
	UsdValue            decimal.Decimal `json:"usd_value"`
	RealizedProfit30d   decimal.Decimal `json:"realized_profit_30d"`
	RealizedProfit      decimal.Decimal `json:"realized_profit"`
	RealizedPnl         decimal.Decimal `json:"realized_pnl"`
	RealizedPnl30d      decimal.Decimal `json:"realized_pnl_30d"`
	UnrealizedProfit    decimal.Decimal `json:"unrealized_profit"`
	UnrealizedPnl       decimal.Decimal `json:"unrealized_pnl"`
	TotalProfit         decimal.Decimal `json:"total_profit"`
	TotalProfitPnl      decimal.Decimal `json:"total_profit_pnl"`
	AvgCost             decimal.Decimal `json:"avg_cost"`
	AvgSold             decimal.Decimal `json:"avg_sold"`
	Buy30d              int             `json:"buy_30d"`
	Sell30d             int             `json:"sell_30d"`
	Sells               int             `json:"sells"`
	Price               decimal.Decimal `json:"price"`
	Cost                decimal.Decimal `json:"cost"`
	PositionPercent     decimal.Decimal `json:"position_percent"`
	LastActiveTimestamp int64           `json:"last_active_timestamp"`
	HistorySoldIncome   decimal.Decimal `json:"history_sold_income"`
	HistoryBoughtCost   decimal.Decimal `json:"history_bought_cost"`
	HistoryBoughtAmount decimal.Decimal `json:"history_bought_amount"`
	HistorySoldAmount   decimal.Decimal `json:"history_sold_amount"`
	StartHoldingAt      int64           `json:"start_holding_at"`
	EndHoldingAt        *int64          `json:"end_holding_at"`
	Liquidity           decimal.Decimal `json:"liquidity"`
	TotalSupply         decimal.Decimal `json:"total_supply"`
	WalletTokenTags     []string        `json:"wallet_token_tags"`
	LastBlock           int64           `json:"last_block"`
}

type WalletHoldings struct {
	Holdings []*WalletHolding `json:"holdings"`
}

type TokenFilter struct {
	MinMarketcap      decimal.Decimal
	MaxMarketcap      decimal.Decimal
	MinHolderCount    int
	MinSwaps1H        int
	MinVolume1H       decimal.Decimal
	MinCreatedMinutes int
	MaxCreatedMinutes int
}

type TokenRank struct {
	ID                       int              `json:"id"`
	Chain                    string           `json:"chain"`
	Address                  string           `json:"address"`
	Symbol                   string           `json:"symbol"`
	Logo                     string           `json:"logo"`
	Price                    decimal.Decimal  `json:"price"`
	PriceChangePercent       decimal.Decimal  `json:"price_change_percent"`
	Swaps                    int              `json:"swaps"`
	Volume                   decimal.Decimal  `json:"volume"`
	Liquidity                decimal.Decimal  `json:"liquidity"`
	MarketCap                decimal.Decimal  `json:"market_cap"`
	HotLevel                 int              `json:"hot_level"`
	PoolCreationTimestamp    int64            `json:"pool_creation_timestamp"` // 使用 int64 处理时间戳
	HolderCount              int              `json:"holder_count"`
	PoolType                 int              `json:"pool_type"`
	PoolTypeStr              string           `json:"pool_type_str"`
	TwitterUsername          string           `json:"twitter_username"`
	Website                  *string          `json:"website"`  // 使用指针以处理可能为 null 的值
	Telegram                 *string          `json:"telegram"` // 使用指针以处理可能为 null 的值
	TotalSupply              int64            `json:"total_supply"`
	OpenTimestamp            int64            `json:"open_timestamp"` // 使用 int64 处理时间戳
	PriceChangePercent1m     decimal.Decimal  `json:"price_change_percent1m"`
	PriceChangePercent5m     decimal.Decimal  `json:"price_change_percent5m"`
	PriceChangePercent1h     decimal.Decimal  `json:"price_change_percent1h"`
	Buys                     int              `json:"buys"`
	Sells                    int              `json:"sells"`
	InitialLiquidity         decimal.Decimal  `json:"initial_liquidity"`
	IsShowAlert              bool             `json:"is_show_alert"`
	Top10HolderRate          decimal.Decimal  `json:"top_10_holder_rate"`
	RenouncedMint            int              `json:"renounced_mint"`
	RenouncedFreezeAccount   int              `json:"renounced_freeze_account"`
	BurnRatio                int              `json:"burn_ratio"`
	BurnStatus               string           `json:"burn_status"`
	Launchpad                string           `json:"launchpad"`
	LaunchpadPlatform        string           `json:"launchpad_platform"`
	ImageDup                 string           `json:"image_dup"`
	DevTokenBurnAmount       *decimal.Decimal `json:"dev_token_burn_amount"` // 使用指针以处理可能为 null 的值
	DevTokenBurnRatio        *decimal.Decimal `json:"dev_token_burn_ratio"`  // 使用指针以处理可能为 null 的值
	DexscrAd                 int              `json:"dexscr_ad"`
	DexscrUpdateLink         int              `json:"dexscr_update_link"`
	CtoFlag                  int              `json:"cto_flag"`
	TwitterChangeFlag        int              `json:"twitter_change_flag"`
	TwitterRenameCount       int              `json:"twitter_rename_count"`
	TwitterDelPostTokenCount int              `json:"twitter_del_post_token_count"`
	TwitterCreateTokenCount  int              `json:"twitter_create_token_count"`
	CreatorTokenStatus       string           `json:"creator_token_status"`
	CreatorClose             bool             `json:"creator_close"`
	Creator                  string           `json:"creator"`
	LaunchpadStatus          int              `json:"launchpad_status"`
	RatTraderAmountRate      decimal.Decimal  `json:"rat_trader_amount_rate"`
	CreatorCreatedInnerCount int              `json:"creator_created_inner_count"`
	CreatorCreatedOpenCount  int              `json:"creator_created_open_count"`
	CreatorCreatedOpenRatio  string           `json:"creator_created_open_ratio"`
	BluechipOwnerPercentage  decimal.Decimal  `json:"bluechip_owner_percentage"`
	RugRatio                 decimal.Decimal  `json:"rug_ratio"`
	SniperCount              int              `json:"sniper_count"`
	SmartDegenCount          int              `json:"smart_degen_count"`
	RenownedCount            int              `json:"renowned_count"`
	IsOG                     *bool            `json:"is_og"` // 使用指针以处理可能为 null 的值
	IsWashTrading            bool             `json:"is_wash_trading"`
}

type TrendingTokens struct {
	Rank []TokenRank `json:"rank"`
}

type gmgnOhlc struct {
	Open   decimal.Decimal `json:"open"`
	Close  decimal.Decimal `json:"close"`
	High   decimal.Decimal `json:"high"`
	Low    decimal.Decimal `json:"low"`
	Time   decimal.Decimal `json:"time"`
	Volume decimal.Decimal `json:"volume"`
}

type gmgnResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type gmgnTokenCandles struct {
	List []gmgnOhlc `json:"list"`
}
