package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fachebot/evm-grid-bot/internal/config"
	"github.com/fachebot/evm-grid-bot/internal/datapi/gmgn"
	"github.com/fachebot/evm-grid-bot/internal/datapi/okxweb3"
	"github.com/fachebot/evm-grid-bot/internal/engine"
	"github.com/fachebot/evm-grid-bot/internal/job"
	"github.com/fachebot/evm-grid-bot/internal/logger"
	"github.com/fachebot/evm-grid-bot/internal/strategy"
	"github.com/fachebot/evm-grid-bot/internal/svc"
	"github.com/fachebot/evm-grid-bot/internal/telebot"
	"github.com/fachebot/evm-grid-bot/internal/utils/evm"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	version     = "dev"
	showVersion = flag.Bool("version", false, "显示版本信息")
	configFile  = flag.String("f", "etc/config.yaml", "the config file")
)

func startAllStrategy(svcCtx *svc.ServiceContext, strategyEngine *engine.StrategyEngine) {
	offset := 0
	const limit = 100

	for {
		data, err := svcCtx.StrategyModel.FindAllActive(context.TODO(), offset, limit)
		if err != nil {
			logger.Fatalf("[startAllStrategy] 加载活跃的策略列表失败, %v", err)
		}

		if len(data) == 0 {
			break
		}

		strategyList := make([]engine.Strategy, 0)
		for _, item := range data {
			s := strategy.NewGridStrategy(svcCtx, item)
			strategyList = append(strategyList, s)
		}

		err = strategyEngine.StartStrategy(strategyList)
		if err != nil {
			logger.Fatalf("[startAllStrategy] 开始策略失败, %v", err)
		}

		offset = offset + len(data)
	}
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("version: %s\n", version)
		return
	}

	// 读取配置文件
	c, err := config.LoadFromFile(*configFile)
	if err != nil {
		logger.Fatalf("读取配置文件失败, %s", err)
	}

	// 创建数据目录
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		err := os.Mkdir("data", 0755)
		if err != nil {
			logger.Fatalf("创建数据目录失败, %s", err)
		}
	}

	// 创建以太坊客户端
	rpcClient, err := rpc.DialContext(context.Background(), c.Chain.RpcUrl)
	if err != nil {
		logger.Fatalf("创建RPC客户端失败, rpcUrl: %s, %v", c.Chain.RpcUrl, err)
	}

	ethClient := ethclient.NewClient(rpcClient)
	chainId, err := ethClient.ChainID(context.Background())
	if err != nil {
		logger.Fatalf("链ID与配置不一致, ChainId: %d, got ChainId: %d", c.Chain.Id, chainId)
	}

	// 查询稳定币有效精度
	tokenMeta, err := evm.GetTokenMeta(context.TODO(), ethClient, c.Chain.StablecoinCA)
	if err != nil {
		logger.Fatalf("查询稳定币有效精度失败, CA: %s, %v", c.Chain.StablecoinCA, err)
	}
	c.Chain.StablecoinSymbol = tokenMeta.Symbol
	c.Chain.StablecoinDecimals = tokenMeta.Decimals

	// 运行K线管理器
	const candles = 329
	const resolution = "1m"
	var quotationSubscriber job.Job
	var klineManager engine.KlineManager
	switch c.Datapi {
	case "okx":
		subscriber, err := okxweb3.NewOkxSubscriber(c.Chain.Id, resolution, c.Sock5Proxy)
		if err != nil {
			logger.Fatalf("创建报价订阅器失败, %s", err)
		}
		subscriber.Start()
		subscriber.WaitUntilConnected()

		okxClient, err := okxweb3.NewClient(c.Chain.Id, c.Sock5Proxy)
		if err != nil {
			logger.Fatalf("创建okx客户端失败, %s", err)
		}
		klineManager = okxweb3.NewKlineManager(okxClient, subscriber, candles)
		klineManager.Start()

		quotationSubscriber = subscriber
	default:
		subscriber, err := gmgn.NewQuotationSubscriber(c.Chain.Id, resolution, nil, c.Sock5Proxy)
		if err != nil {
			logger.Fatalf("创建报价订阅器失败, %s", err)
		}
		subscriber.Start()
		subscriber.WaitUntilConnected()

		gmgnClient, err := gmgn.NewClient(c.Chain.Id, c.Sock5Proxy, c.ZenRows)
		if err != nil {
			logger.Fatalf("创建gmgn客户端失败, %s", err)
		}
		klineManager = gmgn.NewKlineManager(gmgnClient, subscriber, candles)
		klineManager.Start()

		quotationSubscriber = subscriber
	}

	// 运行策略引擎
	strategyEngine := engine.NewStrategyEngine(klineManager)
	strategyEngine.Start()

	// 创建服务上下文
	svcCtx := svc.NewServiceContext(c, strategyEngine, ethClient)

	// 运行订单Keeper
	orderKeeper := job.NewOrderKeeper(svcCtx)
	orderKeeper.Start()

	// 运行机器人服务
	botService, err := telebot.NewTeleBot(svcCtx)
	if err != nil {
		logger.Fatalf("创建机器人服务失败, %s", err)
	}
	botService.Start()

	// 开始所有策略
	startAllStrategy(svcCtx, strategyEngine)

	// 等待程序退出
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	botService.Stop()
	strategyEngine.Stop()
	klineManager.Stop()
	quotationSubscriber.Stop()
	orderKeeper.Stop()

	svcCtx.Close()
	logger.Infof("服务已停止")
}
