package engine

import (
	"context"
	"strings"
	"sync"

	"github.com/fachebot/evm-grid-bot/internal/charts"
	"github.com/fachebot/evm-grid-bot/internal/logger"
)

type Strategy interface {
	ID() string
	TokenAddress() string
	OnTick(ctx context.Context, ohlcs []charts.Ohlc) error
}

type KlineManager interface {
	Start()
	Stop()
	Subscribe(assets []string) error
	Unsubscribe(assets []string) error
	GetOhlcsChan() <-chan charts.TokenOhlcs
}

type StrategyEngine struct {
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}

	klineManager         KlineManager
	mutex                sync.RWMutex
	strategyMap          map[string]Strategy
	tokenStrategyCounter map[string]int
}

func NewStrategyEngine(klineManager KlineManager) *StrategyEngine {
	ctx, cancel := context.WithCancel(context.Background())
	return &StrategyEngine{
		ctx:                  ctx,
		cancel:               cancel,
		klineManager:         klineManager,
		strategyMap:          make(map[string]Strategy),
		tokenStrategyCounter: make(map[string]int),
	}
}

func (engine *StrategyEngine) Stop() {
	if engine.stopChan == nil {
		return
	}

	logger.Infof("[StrategyEngine] 准备停止服务")

	engine.cancel()

	<-engine.stopChan
	close(engine.stopChan)
	engine.stopChan = nil

	logger.Infof("[StrategyEngine] 服务已经停止")
}

func (engine *StrategyEngine) Start() {
	if engine.stopChan != nil {
		return
	}

	engine.stopChan = make(chan struct{})
	logger.Infof("[StrategyEngine] 开始运行服务")
	go engine.run()
}

func (engine *StrategyEngine) StopStrategy(id string) {
	engine.mutex.Lock()
	strategy, ok := engine.strategyMap[id]
	if !ok {
		engine.mutex.Unlock()
		return
	}

	token := strategy.TokenAddress()
	count, ok := engine.tokenStrategyCounter[token]
	if !ok || count < 0 {
		count = 0
	} else if count > 0 {
		count = count - 1
	}
	engine.tokenStrategyCounter[token] = count

	delete(engine.strategyMap, strategy.ID())
	engine.mutex.Unlock()

	if count == 0 {
		err := engine.klineManager.Unsubscribe([]string{token})
		if err != nil {
			logger.Errorf("[StrategyEngine] 取消订阅失败, token: %s, %s", token, err)
		}
	}
}

func (engine *StrategyEngine) StartStrategy(strategyList []Strategy) (err error) {
	if len(strategyList) == 0 {
		return nil
	}

	tokenAddresses := make([]string, 0)
	startedStrategyList := make([]Strategy, 0)

	engine.mutex.RLock()
	for _, strategy := range strategyList {
		_, ok := engine.strategyMap[strategy.ID()]
		if ok {
			continue
		}

		tokenAddresses = append(tokenAddresses, strategy.TokenAddress())
		startedStrategyList = append(startedStrategyList, strategy)
	}
	engine.mutex.RUnlock()

	if len(tokenAddresses) > 0 {
		err = engine.klineManager.Subscribe(tokenAddresses)
		if err != nil {
			return err
		}
	}

	engine.mutex.Lock()
	for _, strategy := range startedStrategyList {
		engine.strategyMap[strategy.ID()] = strategy
	}
	for _, tokenAddress := range tokenAddresses {
		count, ok := engine.tokenStrategyCounter[tokenAddress]
		if !ok {
			count = 1
		} else {
			count = count + 1
		}
		engine.tokenStrategyCounter[tokenAddress] = count
	}
	engine.mutex.Unlock()

	return nil
}

func (engine *StrategyEngine) run() {
	ohlcsChan := engine.klineManager.GetOhlcsChan()

	for {
		select {
		case <-engine.ctx.Done():
			engine.stopChan <- struct{}{}
			return
		case data := <-ohlcsChan:
			strategyList := make([]Strategy, 0)
			engine.mutex.RLock()
			for _, value := range engine.strategyMap {
				strategyList = append(strategyList, value)
			}
			engine.mutex.RUnlock()

			for _, strategy := range strategyList {
				if !strings.EqualFold(strategy.TokenAddress(), data.Token) {
					continue
				}
				err := strategy.OnTick(engine.ctx, data.Ohlcs)
				if err != nil {
					logger.Errorf("[StrategyEngine] 策略执行失败, token: %s, %s", data.Token, err)
				}
			}
		}
	}
}
