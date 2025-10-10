package gmgn

import (
	"context"
	"time"

	"github.com/fachebot/evm-grid-bot/internal/charts"
	"github.com/fachebot/evm-grid-bot/internal/logger"
)

const (
	maxCandelsLimit = 1000
)

type KlineManager struct {
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}

	client         *Client
	subscriber     *QuotationSubscriber
	candles        int
	resolution     time.Duration
	tokenOhlcsMap  map[string][]charts.Ohlc
	tokenOhlcsChan chan charts.TokenOhlcs
}

func NewKlineManager(client *Client, subscriber *QuotationSubscriber, candles int) *KlineManager {
	if candles > maxCandelsLimit {
		candles = maxCandelsLimit
	}

	ctx, cancel := context.WithCancel(context.Background())

	resolution, err := charts.ResolutionToDuration(subscriber.resolution)
	if err != nil {
		logger.Fatalf("[KlineManager] 无效的 resolution 配置, value: %v, %v", subscriber.resolution, err)
	}

	return &KlineManager{
		ctx:           ctx,
		cancel:        cancel,
		client:        client,
		subscriber:    subscriber,
		candles:       candles,
		resolution:    resolution,
		tokenOhlcsMap: make(map[string][]charts.Ohlc),
	}
}

func (m *KlineManager) Stop() {
	if m.stopChan == nil {
		return
	}

	logger.Infof("[KlineManager] 准备停止服务")

	m.cancel()

	<-m.stopChan

	close(m.stopChan)
	m.stopChan = nil

	if m.tokenOhlcsChan != nil {
		close(m.tokenOhlcsChan)
		m.tokenOhlcsChan = nil
	}

	logger.Infof("[KlineManager] 服务已经停止")
}

func (m *KlineManager) Start() {
	if m.stopChan != nil {
		return
	}

	m.stopChan = make(chan struct{})
	logger.Infof("[KlineManager] 开始运行服务")
	go m.run()
}

func (m *KlineManager) Subscribe(assets []string) error {
	return m.subscriber.Subscribe(assets)
}

func (m *KlineManager) Unsubscribe(assets []string) error {
	return nil
}

func (m *KlineManager) GetOhlcsChan() <-chan charts.TokenOhlcs {
	if m.tokenOhlcsChan == nil {
		m.tokenOhlcsChan = make(chan charts.TokenOhlcs, 1024)
	}
	return m.tokenOhlcsChan
}

func (m *KlineManager) run() {
	tickerChan := m.subscriber.GetTickerChan()

	for {
		select {
		case <-m.ctx.Done():
			m.stopChan <- struct{}{}
			return
		case data := <-tickerChan:
			ohlcs, ok := m.updateOhlcs(data)
			if !ok {
				continue
			}

			if m.tokenOhlcsChan != nil {
				select {
				case m.tokenOhlcsChan <- charts.TokenOhlcs{Token: data.Token, Ohlcs: ohlcs}:
				default:
					logger.Warnf("[KlineManager] 分发 Ohlcs 数据, channel 已满. token: %+v", data.Token)
				}
			}
		}
	}
}

func (m *KlineManager) trimOhlcs(ohlcs []charts.Ohlc) []charts.Ohlc {
	if len(ohlcs) <= maxCandelsLimit {
		return ohlcs
	}

	copy(ohlcs, ohlcs[len(ohlcs)-maxCandelsLimit:])
	ohlcs = ohlcs[:maxCandelsLimit]

	return ohlcs
}

func (m *KlineManager) updateOhlcs(data Ticker) ([]charts.Ohlc, bool) {
	ohlcs := m.tokenOhlcsMap[data.Token]

	// 初始化或重置切片
	if data.First {
		ohlcs = ohlcs[:0]
	}

	// 数据加载逻辑封装
	loadData := func() bool {
		newOhlcs, err := m.client.FetchTokenCandles(
			m.ctx, data.Token, data.Ohlc.Time, m.subscriber.resolution, m.candles)
		if err != nil {
			logger.Errorf("[KlineManager] 获取数据失败: %s", err)
			return false
		}
		ohlcs = m.trimOhlcs(newOhlcs)
		m.tokenOhlcsMap[data.Token] = ohlcs
		return true
	}

	shouldReload := func(ohlcs []charts.Ohlc, data Ticker, resolution time.Duration) bool {
		lastTime := ohlcs[len(ohlcs)-1].Time
		return data.Ohlc.Time.Sub(lastTime) > resolution
	}

	// 首次加载或重新加载
	if len(ohlcs) == 0 {
		logger.Infof("[KlineManager] 首次获取K线数据, token: %s", data.Token)
		if loadData() {
			return ohlcs, true
		}
		return nil, false
	} else if shouldReload(ohlcs, data, m.resolution) {
		logger.Infof("[KlineManager] 重新加载K线数据, token: %s, t1: %v, t2: %v, du: %v",
			data.Token, data.Ohlc.Time, ohlcs[len(ohlcs)-1].Time,
			data.Ohlc.Time.Sub(ohlcs[len(ohlcs)-1].Time))
		if loadData() {
			return ohlcs, true
		}
		return nil, false
	}

	// 更新或追加数据
	lastIndex := len(ohlcs) - 1
	if lastIndex == -1 || ohlcs[lastIndex].Time.Before(data.Ohlc.Time) {
		ohlcs = append(ohlcs, data.Ohlc)
	} else {
		ohlcs[lastIndex] = data.Ohlc
	}

	ohlcs = m.trimOhlcs(ohlcs)
	m.tokenOhlcsMap[data.Token] = ohlcs

	return ohlcs, true
}
