package okxweb3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fachebot/evm-grid-bot/internal/charts"
	"github.com/fachebot/evm-grid-bot/internal/config"
	"github.com/fachebot/evm-grid-bot/internal/logger"

	"github.com/gorilla/websocket"
	utls "github.com/refraction-networking/utls"
	"github.com/shopspring/decimal"
	"golang.org/x/net/proxy"
)

// 重连配置
const (
	reconnectInitial = 1 * time.Second  // 初始重连间隔
	reconnectMax     = 30 * time.Second // 最大重连间隔
)

// Ticker K线行情数据
type Ticker struct {
	Token string      // 代币地址
	First bool        // 是否为首条数据
	Ohlc  charts.Ohlc // K线数据
}

// OkxSubscriber OKX Web3行情订阅者
// 通过WebSocket实时订阅代币K线数据
type OkxSubscriber struct {
	ctx      context.Context    // 上下文
	cancel   context.CancelFunc // 取消函数
	stopChan chan struct{}      // 停止信号通道
	url      string             // WebSocket URL

	chainIndex string              // 链索引
	resolution string              // K线周期
	conn       *websocket.Conn     // WebSocket连接
	proxy      config.Sock5Proxy   // SOCKS5代理配置
	reconnect  chan struct{}       // 重连信号通道
	mutex      sync.Mutex          // 互斥锁
	assets     map[string]struct{} // 订阅的代币列表

	tickerChan     chan Ticker    // K线数据通道
	messageCounter map[string]int // 消息计数器
}

// netDialTLSContext 自定义TLS拨号函数
// 支持SOCKS5代理和自定义TLS指纹
func netDialTLSContext(ctx context.Context, network, addr string, sock5Proxy string) (net.Conn, error) {
	serverName := addr
	if host, _, err := net.SplitHostPort(addr); err == nil {
		serverName = host
	}

	// 使用随机TLS指纹
	spec, err := utls.UTLSIdToSpec(RandomClientHelloID())
	if err != nil {
		return nil, err
	}
	// 强制使用HTTP/1.1 ALPN
	for _, ext := range spec.Extensions {
		alpnExt, ok := ext.(*utls.ALPNExtension)
		if !ok {
			continue
		}

		alpnExt.AlpnProtocols = []string{"http/1.1"}
	}

	var conn net.Conn
	// 是否使用SOCKS5代理
	if sock5Proxy == "" {
		conn, err = new(net.Dialer).DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
	} else {
		dialer, err := proxy.SOCKS5(network, sock5Proxy, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}

		conn, err = dialer.Dial(network, addr)
		if err != nil {
			return nil, err
		}
	}

	// 自定义TLS配置
	config := &utls.Config{
		InsecureSkipVerify: true, // 跳过证书验证
		ServerName:         serverName,
	}

	client := utls.UClient(conn, config, utls.HelloCustom)
	if err = client.ApplyPreset(&spec); err != nil {
		return nil, err
	}

	return client, nil
}

// NewOkxSubscriber 创建OKX行情订阅者
// chainId: 链ID
// resolution: K线周期 (如 "1h", "15m")
// proxy: SOCKS5代理配置
func NewOkxSubscriber(chainId int64, resolution string, proxy config.Sock5Proxy) (*OkxSubscriber, error) {
	chainIndex, ok := ChainIdToChainIndex(chainId)
	if !ok {
		return nil, errors.New("unsupported chain")
	}

	ctx, cancel := context.WithCancel(context.Background())
	subscriber := &OkxSubscriber{
		ctx:            ctx,
		cancel:         cancel,
		url:            "wss://wsdexpri.okx.com/ws/v5/ipublic", // OKX WebSocket地址
		chainIndex:     chainIndex,
		resolution:     resolution,
		proxy:          proxy,
		reconnect:      make(chan struct{}, 1),
		assets:         make(map[string]struct{}),
		messageCounter: make(map[string]int),
	}
	return subscriber, nil
}

// Stop 停止订阅服务
func (subscriber *OkxSubscriber) Stop() {
	logger.Infof("[OkxSubscriber] 准备停止服务")

	subscriber.cancel()

	if subscriber.conn != nil {
		subscriber.conn.Close()
	}

	<-subscriber.stopChan

	close(subscriber.stopChan)
	subscriber.stopChan = nil

	if subscriber.tickerChan != nil {
		close(subscriber.tickerChan)
		subscriber.tickerChan = nil
	}

	logger.Infof("[OkxSubscriber] 服务已经停止")
}

// Start 启动订阅服务
func (subscriber *OkxSubscriber) Start() {
	if subscriber.stopChan != nil {
		return
	}

	subscriber.stopChan = make(chan struct{})

	if subscriber.conn == nil {
		logger.Infof("[OkxSubscriber] 开始运行服务")
		go subscriber.run()
	}
}

// WaitUntilConnected 等待连接建立
func (subscriber *OkxSubscriber) WaitUntilConnected() {
	for subscriber.conn == nil {
		time.Sleep(time.Second * 1)
	}
}

// GetTickerChan 获取K线数据通道
func (subscriber *OkxSubscriber) GetTickerChan() <-chan Ticker {
	if subscriber.tickerChan == nil {
		subscriber.tickerChan = make(chan Ticker, 1024)
	}
	return subscriber.tickerChan
}

// Subscribe 订阅代币K线
// assets: 代币地址列表
func (subscriber *OkxSubscriber) Subscribe(assets []string) error {
	// 转换为小写
	for idx, asset := range assets {
		assets[idx] = strings.ToLower(asset)
	}

	// 获取所有已订阅的代币
	allAssets := make([]string, 0)
	subscriber.mutex.Lock()
	for _, asset := range assets {
		subscriber.assets[asset] = struct{}{}
	}
	for asset := range subscriber.assets {
		allAssets = append(allAssets, asset)
	}
	subscriber.mutex.Unlock()

	if len(assets) == 0 {
		return nil
	}
	if subscriber.conn == nil {
		return fmt.Errorf("[OkxSubscriber] 连接未建立")
	}

	logger.Debugf("[OkxSubscriber] 订阅Candle, assets: %+v", assets)

	// 构建订阅参数
	args := make([]map[string]string, 0, len(assets))
	channel := fmt.Sprintf("dex-token-candle%s", subscriber.resolution)
	for _, asset := range allAssets {
		args = append(args, map[string]string{
			"chainId":      subscriber.chainIndex,
			"channel":      channel,
			"tokenAddress": asset,
		})
	}

	payload := map[string]any{
		"op":   "subscribe",
		"args": args,
	}
	err := subscriber.conn.WriteJSON(payload)
	return err
}

// Unsubscribe 取消订阅代币K线
// assets: 代币地址列表
func (subscriber *OkxSubscriber) Unsubscribe(assets []string) error {
	if len(assets) == 0 {
		return nil
	}
	if subscriber.conn == nil {
		return fmt.Errorf("[OkxSubscriber] 连接未建立")
	}

	// 转换为小写
	for idx, asset := range assets {
		assets[idx] = strings.ToLower(asset)
	}

	logger.Debugf("[OkxSubscriber] 取消订阅Candle, assets: %+v", assets)

	// 构建取消订阅参数
	args := make([]map[string]string, 0, len(assets))
	channel := fmt.Sprintf("dex-token-candle%s", subscriber.resolution)
	for _, asset := range assets {
		args = append(args, map[string]string{
			"chainId":      subscriber.chainIndex,
			"channel":      channel,
			"tokenAddress": asset,
		})
	}

	payload := map[string]any{
		"op":   "unsubscribe",
		"args": args,
	}
	err := subscriber.conn.WriteJSON(payload)
	if err == nil {
		subscriber.mutex.Lock()
		for _, asset := range assets {
			delete(subscriber.assets, asset)
		}
		subscriber.mutex.Unlock()
	}

	return err
}

// run 订阅者主循环
func (subscriber *OkxSubscriber) run() {
	subscriber.connect()

	reconnectDelay := reconnectInitial
loop:
	for {
		select {
		case <-subscriber.ctx.Done():
			break loop
		case <-subscriber.reconnect:
			select {
			case <-subscriber.ctx.Done():
				break loop
			case <-time.After(reconnectDelay):
				logger.Infof("[OkxSubscriber] 重新建立连接...")
				subscriber.connect()

				// 指数退避重连
				reconnectDelay *= 2
				if reconnectDelay > reconnectMax {
					reconnectDelay = reconnectMax
				}
			}
		}
	}

	subscriber.stopChan <- struct{}{}
}

// connect 建立WebSocket连接
func (subscriber *OkxSubscriber) connect() {
	// 配置代理
	proxy := ""
	if subscriber.proxy.Enable {
		proxy = fmt.Sprintf("%s:%d", subscriber.proxy.Host, subscriber.proxy.Port)
	}
	dialer := &websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return netDialTLSContext(subscriber.ctx, network, addr, proxy)
		},
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return netDialTLSContext(ctx, network, addr, proxy)
		},
		NetDialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return netDialTLSContext(ctx, network, addr, proxy)
		},
		HandshakeTimeout:  45 * time.Second,
		EnableCompression: true,
	}

	headers := make(http.Header)
	headers.Set("origin", "https://web3.okx.com")
	headers.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")
	headers.Set("accept-language", "zh-CN,zh;q=0.9")
	headers.Set("cache-control", "no-cache")
	headers.Set("pragma", "no-cache")
	headers.Set("accept-encoding", "gzip, deflate, br, zstd")

	// 建立WebSocket连接
	conn, _, err := dialer.Dial(subscriber.url, headers)
	if err != nil {
		logger.Errorf("[OkxSubscriber] 连接失败, %v", err)
		subscriber.scheduleReconnect()
		return
	}

	subscriber.conn = conn
	subscriber.messageCounter = make(map[string]int)
	logger.Infof("[OkxSubscriber] 连接已建立")

	// 重新订阅之前的代币
	assets := make([]string, 0)
	subscriber.mutex.Lock()
	for asset := range subscriber.assets {
		assets = append(assets, asset)
	}
	subscriber.mutex.Unlock()
	subscriber.Subscribe(assets)

	// 启动消息读取协程
	go subscriber.readMessages()
}

// heartbeat 心跳保活
func (subscriber *OkxSubscriber) heartbeat(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// 发送Ping消息
			if err := subscriber.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				logger.Errorf("[OkxSubscriber] 发送心跳消息失败, %v", err)
				return
			}

			// 每20秒发送一次心跳
			duration := time.Second * 20
			timer.Reset(duration)
		case <-ctx.Done():
			return
		}
	}
}

// readMessages 读取WebSocket消息
func (subscriber *OkxSubscriber) readMessages() {
	defer subscriber.conn.Close()

	// 启动心跳协程
	ctx, cancel := context.WithCancel(subscriber.ctx)
	defer cancel()
	go subscriber.heartbeat(ctx)

	for {
		_, message, err := subscriber.conn.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logger.Errorf("[OkxSubscriber] 读取出错, %v", err)
			subscriber.scheduleReconnect()
			return
		}

		logger.Debugf("[OkxSubscriber] 收到新消息, %s", message)

		var payload Message
		if err = json.Unmarshal(message, &payload); err != nil {
			logger.Errorf("[OkxSubscriber] 解析消息失败, message: %s, %v", message, err)
			continue
		}

		// 忽略事件消息
		if payload.Event != "" {
			continue
		}

		channel := fmt.Sprintf("dex-token-candle%s", subscriber.resolution)
		switch payload.GetChannel() {
		case channel:
			var tokenCandles [][]decimal.Decimal
			if err = json.Unmarshal(payload.Data, &tokenCandles); err != nil {
				logger.Errorf("[OkxSubscriber] 解析Candles失败, message: %s, %v", message, err)
				continue
			}

			// 转换为K线数据
			ohlcs := make([]charts.Ohlc, 0, len(tokenCandles))
			for _, data := range tokenCandles {
				if len(data) < 8 {
					logger.Errorf("[OkxSubscriber] Candle数据长度错误, candle: %+v", data)
					continue
				}

				ohlcs = append(ohlcs, charts.Ohlc{
					Open:   data[1],
					Close:  data[2],
					High:   data[3],
					Low:    data[4],
					Time:   time.UnixMilli(data[0].IntPart()),
					Volume: data[6],
				})
			}

			if subscriber.tickerChan != nil {
				tokenAddress := payload.GetTokenAddress()
				for _, ohlc := range ohlcs {
					// 检查是否为该代币的首条数据
					count, ok := subscriber.messageCounter[tokenAddress]
					if !ok {
						count = 0
					}

					ticker := Ticker{
						Token: tokenAddress,
						First: count == 0,
						Ohlc:  ohlc,
					}

					subscriber.messageCounter[tokenAddress] = count + 1

					// 发送到K线数据通道
					select {
					case subscriber.tickerChan <- ticker:
						logger.Debugf("[OkxSubscriber] 分发 Ticker 数据, %+v", ticker)
					default:
						logger.Warnf("[OkxSubscriber] 分发 Ticker 数据, channel 已满. %+v", ticker)
					}
				}
			}
		}
	}
}

// scheduleReconnect 安排重连
func (subscriber *OkxSubscriber) scheduleReconnect() {
	if subscriber.ctx.Err() == nil {
		select {
		case subscriber.reconnect <- struct{}{}:
		default:
		}
	}
}
