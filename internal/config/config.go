package config

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

type Chain struct {
	Id             int64  `yaml:"Id"`
	RpcUrl         string `yaml:"RpcUrl"`
	NativeCurrency struct {
		Symbol   string `yaml:"Symbol"`
		Decimals uint8  `yaml:"Decimals"`
	} `yaml:"NativeCurrency"`
	StablecoinCA       string `yaml:"StablecoinCA"`
	StablecoinSymbol   string `yaml:"-"`
	StablecoinDecimals uint8  `yaml:"-"`
	SlippageBps        int    `yaml:"SlippageBps"`
	DexAggregator      string `yaml:"DexAggregator"`
}

type OkxWeb3 struct {
	Apikey     string `yaml:"Apikey"`
	Secretkey  string `yaml:"Secretkey"`
	Passphrase string `yaml:"Passphrase"`
}

type DeepSeek struct {
	Apikey string `yaml:"Apikey"`
}

type ZenRows struct {
	Apikey              string `yaml:"Apikey"`
	FetchTokenCandles   bool   `yaml:"FetchTokenCandles"`
	FetchTokenHolders   bool   `yaml:"FetchTokenHolders"`
	FetchWalletHoldings bool   `yaml:"FetchWalletHoldings"`
}

type Sock5Proxy struct {
	Host   string `yaml:"Host"`
	Port   int32  `yaml:"Port"`
	Enable bool   `yaml:"Enable"`
}

type LoreFilter struct {
	MinHolder int `yaml:"MinHolder"`
}

type TelegramBot struct {
	Debug     bool    `yaml:"Debug"`
	ApiToken  string  `yaml:"ApiToken"`
	WhiteList []int64 `yaml:"WhiteList"`
}

func (c *TelegramBot) IsWhiteListUser(userId int64) bool {
	if len(c.WhiteList) == 0 {
		return true
	}
	return slices.Contains(c.WhiteList, userId)
}

type DefaultGridSettings struct {
	OrderSize             decimal.Decimal `yaml:"OrderSize"`
	MaxGridLimit          int             `yaml:"MaxGridLimit"`
	StopLossExit          decimal.Decimal `yaml:"StopLossExit"`
	TakeProfitExit        decimal.Decimal `yaml:"TakeProfitExit"`
	TakeProfitRatio       decimal.Decimal `yaml:"TakeProfitRatio"`
	EnableAutoExit        bool            `yaml:"EnableAutoExit"`
	LastKlineVolume       decimal.Decimal `yaml:"LastKlineVolume"`
	FiveKlineVolume       decimal.Decimal `yaml:"FiveKlineVolume"`
	GlobalTakeProfitRatio decimal.Decimal `yaml:"GlobalTakeProfitRatio"`
	DropOn                bool            `yaml:"DropOn"`
	CandlesToCheck        int             `yaml:"CandlesToCheck"`
	DropThreshold         decimal.Decimal `yaml:"DropThreshold"`
}

func (c *DefaultGridSettings) Validate() error {
	if c.MaxGridLimit <= 0 {
		c.MaxGridLimit = 10
	}
	if c.OrderSize.LessThanOrEqual(decimal.Zero) {
		c.OrderSize = decimal.NewFromInt(40)
	}
	if c.TakeProfitRatio.LessThanOrEqual(decimal.Zero) {
		c.TakeProfitRatio = decimal.NewFromFloat(3.5)
	}

	if c.MaxGridLimit <= 0 {
		c.MaxGridLimit = 10
	}
	if c.OrderSize.LessThanOrEqual(decimal.Zero) {
		c.OrderSize = decimal.NewFromInt(40)
	}
	if c.TakeProfitRatio.LessThanOrEqual(decimal.Zero) {
		c.TakeProfitRatio = decimal.NewFromFloat(3.5)
	}

	if c.GlobalTakeProfitRatio.LessThan(decimal.Zero) {
		return errors.New("GlobalTakeProfitRatio 不能小于0")
	}

	if c.CandlesToCheck < 0 {
		c.CandlesToCheck = 0
	}
	if c.DropThreshold.LessThan(decimal.Zero) {
		c.DropThreshold = decimal.Zero
	}

	return nil
}

type QuickStartSettings struct {
	OrderSize             decimal.Decimal `yaml:"OrderSize"`
	MaxGridLimit          int             `yaml:"MaxGridLimit"`
	StopLossExit          decimal.Decimal `yaml:"StopLossExit"`
	TakeProfitExit        decimal.Decimal `yaml:"TakeProfitExit"`
	TakeProfitRatio       decimal.Decimal `yaml:"TakeProfitRatio"`
	EnableAutoExit        bool            `yaml:"EnableAutoExit"`
	LastKlineVolume       decimal.Decimal `yaml:"LastKlineVolume"`
	FiveKlineVolume       decimal.Decimal `yaml:"FiveKlineVolume"`
	UpperPriceBound       decimal.Decimal `yaml:"UpperPriceBound"`
	LowerPriceBound       decimal.Decimal `yaml:"LowerPriceBound"`
	GlobalTakeProfitRatio decimal.Decimal `yaml:"GlobalTakeProfitRatio"`
	DropOn                bool            `yaml:"DropOn"`
	CandlesToCheck        int             `yaml:"CandlesToCheck"`
	DropThreshold         decimal.Decimal `yaml:"DropThreshold"`
}

type TokenRequirements struct {
	MinMarketCap       decimal.Decimal `yaml:"MinMarketCap"`
	MinHolderCount     int             `yaml:"MinHolderCount"`
	MinTokenAgeMinutes int             `yaml:"MinTokenAgeMinutes"`
	MaxTokenAgeMinutes int             `yaml:"MaxTokenAgeMinutes"`
}

type Config struct {
	Chain               Chain               `yaml:"Chain"`
	Datapi              string              `yaml:"Datapi"`
	OkxWeb3             OkxWeb3             `yaml:"OkxWeb3"`
	ZenRows             ZenRows             `yaml:"ZenRows"`
	DeepSeek            DeepSeek            `yaml:"DeepSeek"`
	Sock5Proxy          Sock5Proxy          `yaml:"Sock5Proxy"`
	LoreFilter          LoreFilter          `yaml:"LoreFilter"`
	TelegramBot         TelegramBot         `yaml:"TelegramBot"`
	DefaultGridSettings DefaultGridSettings `yaml:"DefaultGridSettings"`
	QuickStartSettings  QuickStartSettings  `yaml:"QuickStartSettings"`
	TokenRequirements   TokenRequirements   `yaml:"TokenRequirements"`
}

func LoadFromFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var c Config
	err = yaml.Unmarshal([]byte(data), &c)
	if err != nil {
		return nil, err
	}

	if err = c.DefaultGridSettings.Validate(); err != nil {
		return nil, fmt.Errorf("DefaultGridSettings配置错误: %w", err)
	}

	if c.Datapi != "gmgn" && c.Datapi != "okx" {
		return nil, errors.New("Datapi配置枚举值范围: gmgn/okx")
	}

	return &c, nil
}
