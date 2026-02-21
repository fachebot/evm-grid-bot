package okxweb3

import (
	"context"
	"time"

	"github.com/fachebot/evm-grid-bot/internal/charts"
	"github.com/fachebot/evm-grid-bot/internal/logger"
)

// 最大K线数量限制
const (
	maxCandlesLimit = 1000
)

// KlineManager K线数据管理器
// 负责管理代币K线数据的订阅、缓存和分发
type KlineManager struct {
	ctx      context.Context    // 上下文
	cancel   context.CancelFunc // 取消函数
	stopChan chan struct{}      // 停止信号通道

	client         *Client                  // OKX API客户端
	subscriber     *OkxSubscriber           // WebSocket订阅者
	candles        int                      // K线数量
	resolution     time.Duration            // K线周期
	tokenOhlcsMap  map[string][]charts.Ohlc // 代币K线数据缓存
	tokenOhlcsChan chan charts.TokenOhlcs   // K线数据分发通道
}

// NewKlineManager 创建K线管理器
// client: OKX API客户端
// subscriber: WebSocket订阅者
// candles: 初始K线数量
func NewKlineManager(client *Client, subscriber *OkxSubscriber, candles int) *KlineManager {
	if candles > maxCandlesLimit {
		candles = maxCandlesLimit
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 解析K线周期
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

// Stop 停止K线管理器
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

// Start 启动K线管理器
func (m *KlineManager) Start() {
	if m.stopChan != nil {
		return
	}

	m.stopChan = make(chan struct{})
	logger.Infof("[KlineManager] 开始运行服务")
	go m.run()
}

// Subscribe 订阅代币K线
// assets: 代币地址列表
func (m *KlineManager) Subscribe(assets []string) error {
	return m.subscriber.Subscribe(assets)
}

// Unsubscribe 取消订阅代币K线
// assets: 代币地址列表
func (m *KlineManager) Unsubscribe(assets []string) error {
	return m.subscriber.Unsubscribe(assets)
}

// GetOhlcsChan 获取K线数据通道
func (m *KlineManager) GetOhlcsChan() <-chan charts.TokenOhlcs {
	if m.tokenOhlcsChan == nil {
		m.tokenOhlcsChan = make(chan charts.TokenOhlcs, 1024)
	}
	return m.tokenOhlcsChan
}

// run K线管理器主循环
func (m *KlineManager) run() {
	// 获取WebSocket K线数据通道
	tickerChan := m.subscriber.GetTickerChan()

	for {
		select {
		case <-m.ctx.Done():
			m.stopChan <- struct{}{}
			return
		case data := <-tickerChan:
			// 更新K线数据
			ohlcs, ok := m.updateOhlcs(data)
			if !ok {
				continue
			}

			// 分发K线数据
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

// trimOhlcs 裁剪K线数据
// 保持最新的maxCandlesLimit条数据
func (m *KlineManager) trimOhlcs(ohlcs []charts.Ohlc) []charts.Ohlc {
	if len(ohlcs) <= maxCandlesLimit {
		return ohlcs
	}

	copy(ohlcs, ohlcs[len(ohlcs)-maxCandlesLimit:])
	ohlcs = ohlcs[:maxCandlesLimit]

	return ohlcs
}

// updateOhlcs 更新K线数据
// data: WebSocket推送的K线数据
// 返回: 更新后的K线数据, 是否成功
func (m *KlineManager) updateOhlcs(data Ticker) ([]charts.Ohlc, bool) {
	ohlcs := m.tokenOhlcsMap[data.Token]

	// 首次收到数据或重置
	if data.First {
		ohlcs = ohlcs[:0]
	}

	// 加载数据函数
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

	// 判断是否需要重新加载
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
		// 追加新数据
		ohlcs = append(ohlcs, data.Ohlc)
	} else {
		// 更新最新数据
		ohlcs[lastIndex] = data.Ohlc
	}

	// 裁剪数据并缓存
	ohlcs = m.trimOhlcs(ohlcs)
	m.tokenOhlcsMap[data.Token] = ohlcs

	return ohlcs, true
}
