package svc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/fachebot/evm-grid-bot/internal/cache"
	"github.com/fachebot/evm-grid-bot/internal/config"
	"github.com/fachebot/evm-grid-bot/internal/datapi/gmgn"
	"github.com/fachebot/evm-grid-bot/internal/datapi/okxweb3"
	"github.com/fachebot/evm-grid-bot/internal/engine"
	"github.com/fachebot/evm-grid-bot/internal/ent"
	"github.com/fachebot/evm-grid-bot/internal/eth"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/model"
	"github.com/fachebot/evm-grid-bot/internal/utils"

	"github.com/ethereum/go-ethereum/ethclient"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/proxy"
)

type ServiceContext struct {
	Config         *config.Config
	HashEncoder    *utils.HashEncoder
	Engine         *engine.StrategyEngine
	DbClient       *ent.Client
	BotApi         *tgbotapi.BotAPI
	BotUserInfo    *tgbotapi.User
	OkxClient      *okxweb3.Client
	GmgnClient     *gmgn.Client
	TransportProxy *http.Transport
	EthClient      *ethclient.Client
	MessageCache   *cache.MessageCache
	TokenMetaCache *cache.TokenMetaCache
	GridModel      *model.GridModel
	OrderModel     *model.OrderModel
	SettingsModel  *model.SettingsModel
	StrategyModel  *model.StrategyModel
	WalletModel    *model.WalletModel
	NonceManager   *eth.NonceManager
}

func NewServiceContext(c *config.Config, strategyEngine *engine.StrategyEngine, ethClient *ethclient.Client) *ServiceContext {
	// 创建hash编码器
	salt := os.Getenv("GRIDBOT_HASH_SALT")
	if salt == "" {
		salt = "8wKzxf51vQJT5n=bM6e?z)6B]XiDXcMdE]=>GiXm"
		logger.Debugf("环境变量 GRIDBOT_HASH_SALT 未设置")
	}
	hashEncoder, err := utils.NewHashEncoder(salt)
	if err != nil {
		logger.Fatalf("创建Hash编码器失败, %v", err)
	}

	// 创建数据库连接
	client, err := ent.Open("sqlite3", "file:data/sqlite.db?mode=rwc&_journal_mode=WAL&_fk=1")
	if err != nil {
		logger.Fatalf("打开数据库失败, %v", err)
	}
	if err := client.Schema.Create(context.Background()); err != nil {
		logger.Fatalf("创建数据库Schema失败, %v", err)
	}

	// 创建SOCKS5代理
	var transportProxy *http.Transport
	if c.Sock5Proxy.Enable {
		socks5Proxy := fmt.Sprintf("%s:%d", c.Sock5Proxy.Host, c.Sock5Proxy.Port)
		dialer, err := proxy.SOCKS5("tcp", socks5Proxy, nil, proxy.Direct)
		if err != nil {
			logger.Fatalf("创建SOCKS5代理失败, %v", err)
		}

		transportProxy = &http.Transport{
			Dial:            dialer.Dial,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// 创建OKX客户端
	okxClient, err := okxweb3.NewClient(c.Chain.Id, c.Sock5Proxy)
	if err != nil {
		logger.Fatalf("创建okx客户端失败, %s", err)
	}

	// 创建GMGN客户端
	gmgnClient, err := gmgn.NewClient(c.Chain.Id, c.Sock5Proxy, c.ZenRows)
	if err != nil {
		logger.Fatalf("创建gmgn客户端失败, %s", err)
	}

	// 创建电报机器人
	tgHttpClient := new(http.Client)
	if transportProxy != nil {
		tgHttpClient.Transport = transportProxy
	}
	botApi, err := tgbotapi.NewBotAPIWithClient(c.TelegramBot.ApiToken, tgbotapi.APIEndpoint, tgHttpClient)
	if err != nil {
		logger.Fatalf("创建电报机器人失败, %v", err)
	}
	botApi.Debug = c.TelegramBot.Debug

	botUserInfo, err := botApi.GetMe()
	if err != nil {
		logger.Fatalf("获取电报机器人信息失败, %v", err)
	}

	svcCtx := &ServiceContext{
		Config:         c,
		HashEncoder:    hashEncoder,
		Engine:         strategyEngine,
		DbClient:       client,
		BotApi:         botApi,
		BotUserInfo:    &botUserInfo,
		EthClient:      ethClient,
		OkxClient:      okxClient,
		GmgnClient:     gmgnClient,
		TransportProxy: transportProxy,
		MessageCache:   cache.NewMessageCache(),
		TokenMetaCache: cache.NewTokenMetaCache(ethClient),
		GridModel:      model.NewGridModel(client.Grid),
		OrderModel:     model.NewOrderModel(client.Order),
		SettingsModel:  model.NewSettingsModel(client.Settings),
		StrategyModel:  model.NewStrategyModel(client.Strategy),
		WalletModel:    model.NewWalletModel(client.Wallet),
		NonceManager:   eth.NewNonceManager(client, ethClient),
	}

	return svcCtx
}

func (svcCtx *ServiceContext) Close() {
	if err := svcCtx.DbClient.Close(); err != nil {
		logger.Errorf("关闭数据库失败, %v", err)
	}
}
