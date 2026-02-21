package gmgn

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

// TokenTransfer 代币转账记录
type TokenTransfer struct {
	Name      *string `json:"name"`      // 代币名称
	Address   string  `json:"address"`   // 代币地址
	Timestamp int64   `json:"timestamp"` // 时间戳
	TxHash    string  `json:"tx_hash"`   // 交易哈希
	Type      string  `json:"type"`      // 转账类型
}

// NativeTransfer 原生代币转账记录
type NativeTransfer struct {
	Name        *string `json:"name"`         // 代币名称
	FromAddress string  `json:"from_address"` // 转出地址
	Timestamp   int64   `json:"timestamp"`    // 时间戳
}

// HolderInfo 代币持有者信息
type HolderInfo struct {
	Address              string           `json:"address"`                // 持有者地址
	AccountAddress       string           `json:"account_address"`        // 账户地址
	AddrType             int              `json:"addr_type"`              // 地址类型
	AmountCur            decimal.Decimal  `json:"amount_cur"`             // 当前数量
	UsdValue             decimal.Decimal  `json:"usd_value"`              // USD价值
	CostCur              decimal.Decimal  `json:"cost_cur"`               // 成本
	SellAmountCur        decimal.Decimal  `json:"sell_amount_cur"`        // 卖出数量
	SellAmountPercentage decimal.Decimal  `json:"sell_amount_percentage"` // 卖出占比
	SellVolumeCur        decimal.Decimal  `json:"sell_volume_cur"`        // 卖出金额
	BuyVolumeCur         decimal.Decimal  `json:"buy_volume_cur"`         // 买入金额
	BuyAmountCur         decimal.Decimal  `json:"buy_amount_cur"`         // 买入数量
	NetflowUsd           decimal.Decimal  `json:"netflow_usd"`            // USD净流入
	NetflowAmount        decimal.Decimal  `json:"netflow_amount"`         // 数量净流入
	BuyTxCountCur        int              `json:"buy_tx_count_cur"`       // 买入交易数
	SellTxCountCur       int              `json:"sell_tx_count_cur"`      // 卖出交易数
	WalletTagV2          string           `json:"wallet_tag_v2"`          // 钱包标签
	NativeBalance        string           `json:"native_balance"`         // 原生代币余额
	Balance              decimal.Decimal  `json:"balance"`                // 余额
	Profit               decimal.Decimal  `json:"profit"`                 // 盈亏
	RealizedProfit       decimal.Decimal  `json:"realized_profit"`        // 已实现盈亏
	ProfitChange         *decimal.Decimal `json:"profit_change"`          // 盈亏变化
	AmountPercentage     decimal.Decimal  `json:"amount_percentage"`      // 数量占比
	UnrealizedProfit     decimal.Decimal  `json:"unrealized_profit"`      // 未实现盈亏
	UnrealizedPnl        *decimal.Decimal `json:"unrealized_pnl"`         // 未实现盈亏率
	AvgCost              *decimal.Decimal `json:"avg_cost"`               // 平均成本
	AvgSold              *decimal.Decimal `json:"avg_sold"`               // 平均卖出价
	AccuAmount           decimal.Decimal  `json:"accu_amount"`            // 累计数量
	AccuCost             decimal.Decimal  `json:"accu_cost"`              // 累计成本
	Cost                 decimal.Decimal  `json:"cost"`                   // 成本
	TotalCost            decimal.Decimal  `json:"total_cost"`             // 总成本
	TransferIn           bool             `json:"transfer_in"`            // 是否转入
	IsNew                bool             `json:"is_new"`                 // 是否新增
	IsSuspicious         bool             `json:"is_suspicious"`          // 是否可疑
	IsOnCurve            bool             `json:"is_on_curve"`            // 是否在Curve上
	StartHoldingAt       int64            `json:"start_holding_at"`       // 开始持有时间
	EndHoldingAt         *int64           `json:"end_holding_at"`         // 结束持有时间
	LastActiveTimestamp  int64            `json:"last_active_timestamp"`  // 最后活跃时间
	NativeTransfer       NativeTransfer   `json:"native_transfer"`        // 原生代币转账
	TokenTransfer        TokenTransfer    `json:"token_transfer"`         // 代币转账
	TokenTransferIn      TokenTransfer    `json:"token_transfer_in"`      // 转入代币
	TokenTransferOut     TokenTransfer    `json:"token_transfer_out"`     // 转出代币
	Tags                 []string         `json:"tags"`                   // 标签列表
	MakerTokenTags       []string         `json:"maker_token_tags"`       // 创世代币标签
	Name                 string           `json:"name"`                   // 名称
	Avatar               string           `json:"avatar"`                 // 头像
	TwitterUsername      *string          `json:"twitter_username"`       // Twitter用户名
	TwitterName          *string          `json:"twitter_name"`           // Twitter名称
	CreatedAt            int64            `json:"created_at"`             // 创建时间
}

// HolderInfoList 代币持有者列表
type HolderInfoList struct {
	List []*HolderInfo `json:"list"` // 持有者列表
}

// TokenHolding 代币持仓信息
type TokenHolding struct {
	Address           string          `json:"address"`            // 代币地址
	TokenAddress      string          `json:"token_address"`      // 代币合约地址
	Symbol            string          `json:"symbol"`             // 代币符号
	Name              string          `json:"name"`               // 代币名称
	Decimals          int             `json:"decimals"`           // 代币精度
	Logo              string          `json:"logo"`               // 代币Logo
	PriceChange6h     decimal.Decimal `json:"price_change_6h"`    // 6小时价格变化
	IsShowAlert       bool            `json:"is_show_alert"`      // 是否显示提醒
	IsHoneypot        *bool           `json:"is_honeypot"`        // 是否蜜罐代币
	CreationTimestamp int64           `json:"creation_timestamp"` // 创建时间戳
	OpenTimestamp     int64           `json:"open_timestamp"`     // 开放时间戳
}

// WalletHolding 钱包持仓信息
type WalletHolding struct {
	Token               TokenHolding    `json:"token"`                 // 代币信息
	Balance             decimal.Decimal `json:"balance"`               // 持仓数量
	UsdValue            decimal.Decimal `json:"usd_value"`             // USD价值
	RealizedProfit30d   decimal.Decimal `json:"realized_profit_30d"`   // 30天已实现盈亏
	RealizedProfit      decimal.Decimal `json:"realized_profit"`       // 已实现盈亏
	RealizedPnl         decimal.Decimal `json:"realized_pnl"`          // 已实现盈亏率
	RealizedPnl30d      decimal.Decimal `json:"realized_pnl_30d"`      // 30天已实现盈亏率
	UnrealizedProfit    decimal.Decimal `json:"unrealized_profit"`     // 未实现盈亏
	UnrealizedPnl       decimal.Decimal `json:"unrealized_pnl"`        // 未实现盈亏率
	TotalProfit         decimal.Decimal `json:"total_profit"`          // 总盈亏
	TotalProfitPnl      decimal.Decimal `json:"total_profit_pnl"`      // 总盈亏率
	AvgCost             decimal.Decimal `json:"avg_cost"`              // 平均成本
	AvgSold             decimal.Decimal `json:"avg_sold"`              // 平均卖出价
	Buy30d              int             `json:"buy_30d"`               // 30天买入数
	Sell30d             int             `json:"sell_30d"`              // 30天卖出数
	Sells               int             `json:"sells"`                 // 卖出次数
	Price               decimal.Decimal `json:"price"`                 // 价格
	Cost                decimal.Decimal `json:"cost"`                  // 成本
	PositionPercent     decimal.Decimal `json:"position_percent"`      // 仓位占比
	LastActiveTimestamp int64           `json:"last_active_timestamp"` // 最后活跃时间
	HistorySoldIncome   decimal.Decimal `json:"history_sold_income"`   // 历史卖出收入
	HistoryBoughtCost   decimal.Decimal `json:"history_bought_cost"`   // 历史买入成本
	HistoryBoughtAmount decimal.Decimal `json:"history_bought_amount"` // 历史买入数量
	HistorySoldAmount   decimal.Decimal `json:"history_sold_amount"`   // 历史卖出数量
	StartHoldingAt      int64           `json:"start_holding_at"`      // 开始持有时间
	EndHoldingAt        *int64          `json:"end_holding_at"`        // 结束持有时间
	Liquidity           decimal.Decimal `json:"liquidity"`             // 流动性
	TotalSupply         decimal.Decimal `json:"total_supply"`          // 总供应量
	WalletTokenTags     []string        `json:"wallet_token_tags"`     // 钱包代币标签
	LastBlock           int64           `json:"last_block"`            // 最后区块
}

// WalletHoldings 钱包持仓列表
type WalletHoldings struct {
	Holdings []*WalletHolding `json:"holdings"` // 持仓列表
}

// TokenFilter 代币过滤条件
type TokenFilter struct {
	MinMarketcap      decimal.Decimal // 最小市值
	MaxMarketcap      decimal.Decimal // 最大市值
	MinHolderCount    int             // 最小持币人数
	MinSwaps1H        int             // 最小1小时交易次数
	MinVolume1H       decimal.Decimal // 最小1小时交易量
	MinCreatedMinutes int             // 最小创建时间(分钟)
	MaxCreatedMinutes int             // 最大创建时间(分钟)
}

// TokenRank 代币排名信息
type TokenRank struct {
	ID                       int              `json:"id"`                           // ID
	Chain                    string           `json:"chain"`                        // 链
	Address                  string           `json:"address"`                      // 地址
	Symbol                   string           `json:"symbol"`                       // 符号
	Logo                     string           `json:"logo"`                         // Logo
	Price                    decimal.Decimal  `json:"price"`                        // 价格
	PriceChangePercent       decimal.Decimal  `json:"price_change_percent"`         // 价格变化百分比
	Swaps                    int              `json:"swaps"`                        // 交易次数
	Volume                   decimal.Decimal  `json:"volume"`                       // 交易量
	Liquidity                decimal.Decimal  `json:"liquidity"`                    // 流动性
	MarketCap                decimal.Decimal  `json:"market_cap"`                   // 市值
	HotLevel                 int              `json:"hot_level"`                    // 热门等级
	PoolCreationTimestamp    int64            `json:"pool_creation_timestamp"`      // 资金池创建时间
	HolderCount              int              `json:"holder_count"`                 // 持币人数
	PoolType                 int              `json:"pool_type"`                    // 资金池类型
	PoolTypeStr              string           `json:"pool_type_str"`                // 资金池类型字符串
	TwitterUsername          string           `json:"twitter_username"`             // Twitter用户名
	Website                  *string          `json:"website"`                      // 网站
	Telegram                 *string          `json:"telegram"`                     // 电报
	TotalSupply              int64            `json:"total_supply"`                 // 总供应量
	OpenTimestamp            int64            `json:"open_timestamp"`               // 开放时间
	PriceChangePercent1m     decimal.Decimal  `json:"price_change_percent1m"`       // 1分钟价格变化
	PriceChangePercent5m     decimal.Decimal  `json:"price_change_percent5m"`       // 5分钟价格变化
	PriceChangePercent1h     decimal.Decimal  `json:"price_change_percent1h"`       // 1小时价格变化
	Buys                     int              `json:"buys"`                         // 买入次数
	Sells                    int              `json:"sells"`                        // 卖出次数
	InitialLiquidity         decimal.Decimal  `json:"initial_liquidity"`            // 初始流动性
	IsShowAlert              bool             `json:"is_show_alert"`                // 是否显示提醒
	Top10HolderRate          decimal.Decimal  `json:"top_10_holder_rate"`           // 前10持有率
	RenouncedMint            int              `json:"renounced_mint"`               // 是否放弃铸造权限
	RenouncedFreezeAccount   int              `json:"renounced_freeze_account"`     // 是否放弃冻结权限
	BurnRatio                int              `json:"burn_ratio"`                   // 燃烧比例
	BurnStatus               string           `json:"burn_status"`                  // 燃烧状态
	Launchpad                string           `json:"launchpad"`                    // 发射台
	LaunchpadPlatform        string           `json:"launchpad_platform"`           // 发射台平台
	ImageDup                 string           `json:"image_dup"`                    // 图片重复
	DevTokenBurnAmount       *decimal.Decimal `json:"dev_token_burn_amount"`        // 开发者代币燃烧数量
	DevTokenBurnRatio        *decimal.Decimal `json:"dev_token_burn_ratio"`         // 开发者代币燃烧比例
	DexscrAd                 int              `json:"dexscr_ad"`                    // 广告标识
	DexscrUpdateLink         int              `json:"dexscr_update_link"`           // 更新链接标识
	CtoFlag                  int              `json:"cto_flag"`                     // CTO标识
	TwitterChangeFlag        int              `json:"twitter_change_flag"`          // Twitter变更标识
	TwitterRenameCount       int              `json:"twitter_rename_count"`         // Twitter重命名次数
	TwitterDelPostTokenCount int              `json:"twitter_del_post_token_count"` // Twitter删除帖子数
	TwitterCreateTokenCount  int              `json:"twitter_create_token_count"`   // Twitter创建代币数
	CreatorTokenStatus       string           `json:"creator_token_status"`         // 创建者代币状态
	CreatorClose             bool             `json:"creator_close"`                // 创建者关闭
	Creator                  string           `json:"creator"`                      // 创建者地址
	LaunchpadStatus          int              `json:"launchpad_status"`             // 发射台状态
	RatTraderAmountRate      decimal.Decimal  `json:"rat_trader_amount_rate"`       // 老鼠仓比例
	CreatorCreatedInnerCount int              `json:"creator_created_inner_count"`  // 创建者内部创建数
	CreatorCreatedOpenCount  int              `json:"creator_created_open_count"`   // 创建者公开创建数
	CreatorCreatedOpenRatio  string           `json:"creator_created_open_ratio"`   // 创建者公开创建比例
	BluechipOwnerPercentage  decimal.Decimal  `json:"bluechip_owner_percentage"`    // 蓝筹持有比例
	RugRatio                 decimal.Decimal  `json:"rug_ratio"`                    //  rug比例
	SniperCount              int              `json:"sniper_count"`                 // 狙击手数量
	SmartDegenCount          int              `json:"smart_degen_count"`            // 聪明开发者数量
	RenownedCount            int              `json:"renowned_count"`               // 知名数量
	IsOG                     *bool            `json:"is_og"`                        // 是否OG
	IsWashTrading            bool             `json:"is_wash_trading"`              // 是否洗售交易
}

// TrendingTokens 热门代币列表
type TrendingTokens struct {
	Rank []TokenRank `json:"rank"` // 代币排名列表
}

// gmgnOhlc GMGN K线数据
type gmgnOhlc struct {
	Open   decimal.Decimal `json:"open"`   // 开盘价
	Close  decimal.Decimal `json:"close"`  // 收盘价
	High   decimal.Decimal `json:"high"`   // 最高价
	Low    decimal.Decimal `json:"low"`    // 最低价
	Time   decimal.Decimal `json:"time"`   // 时间
	Volume decimal.Decimal `json:"volume"` // 成交量
}

// gmgnResponse GMGN API响应
type gmgnResponse struct {
	Code int             `json:"code"` // 错误码
	Msg  string          `json:"msg"`  // 错误消息
	Data json.RawMessage `json:"data"` // 数据
}

// gmgnTokenCandles GMGN代币K线数据
type gmgnTokenCandles struct {
	List []gmgnOhlc `json:"list"` // K线列表
}
