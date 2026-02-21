package gmgn

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
	"github.com/fachebot/evm-grid-bot/internal/utils"

	"github.com/google/uuid"
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

// channelMessage WebSocket通道消息
type channelMessage struct {
	Channel string          `json:"channel"` // 通道名称
	Data    json.RawMessage `json:"data"`    // 数据内容
}

// klineChannelData K线通道数据
type klineChannelData struct {
	N string          `json:"n"` // 名称
	A string          `json:"a"` // 代币地址
	I string          `json:"i"` // 间隔
	O decimal.Decimal `json:"o"` // 开盘价
	H decimal.Decimal `json:"h"` // 最高价
	L decimal.Decimal `json:"l"` // 最低价
	C decimal.Decimal `json:"c"` // 收盘价
	V decimal.Decimal `json:"v"` // 成交量
	T int64           `json:"t"` // 时间戳
}

// QuotationSubscriber GMGN行情订阅者
// 通过WebSocket实时订阅代币K线数据
type QuotationSubscriber struct {
	ctx      context.Context    // 上下文
	cancel   context.CancelFunc // 取消函数
	stopChan chan struct{}      // 停止信号通道

	conn *websocket.Conn // WebSocket连接
	url  string          // WebSocket URL

	chain          string            // 链名称
	resolution     string            // K线周期
	tokenAddresses sync.Map          // 代币地址列表(线程安全)
	proxy          config.Sock5Proxy // SOCKS5代理配置
	reconnect      chan struct{}     // 重连信号通道

	tickerChan     chan Ticker    // K线数据通道
	messageCounter map[string]int // 消息计数器
}

// NewQuotationSubscriber 创建GMGN行情订阅者
// chainId: 链ID
// resolution: K线周期 (如 "1h", "15m")
// tokenAddresses: 初始订阅的代币地址列表
// proxy: SOCKS5代理配置
func NewQuotationSubscriber(
	chainId int64,
	resolution string,
	tokenAddresses []string,
	proxy config.Sock5Proxy,
) (*QuotationSubscriber, error) {
	chain, ok := ChainIdToChainName(chainId)
	if !ok {
		return nil, errors.New("unsupported chain")
	}

	ctx, cancel := context.WithCancel(context.Background())
	subscriber := &QuotationSubscriber{
		url:            "wss://ws.gmgn.ai/quotation", // GMGN WebSocket行情地址
		chain:          chain,
		resolution:     resolution,
		proxy:          proxy,
		reconnect:      make(chan struct{}, 1),
		ctx:            ctx,
		cancel:         cancel,
		messageCounter: make(map[string]int),
	}

	// 初始化代币地址列表
	for _, tokenAddress := range tokenAddresses {
		subscriber.tokenAddresses.Store(tokenAddress, true)
	}
	return subscriber, nil
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

// Stop 停止订阅服务
func (subscriber *QuotationSubscriber) Stop() {
	logger.Infof("[QuotationSubscriber] 准备停止服务")

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

	logger.Infof("[QuotationSubscriber] 服务已经停止")
}

// Start 启动订阅服务
func (subscriber *QuotationSubscriber) Start() {
	if subscriber.stopChan != nil {
		return
	}

	subscriber.stopChan = make(chan struct{})

	if subscriber.conn == nil {
		logger.Infof("[QuotationSubscriber] 开始运行服务")
		go subscriber.run()
	}
}

// WaitUntilConnected 等待连接建立
func (subscriber *QuotationSubscriber) WaitUntilConnected() {
	for subscriber.conn == nil {
		time.Sleep(time.Second * 1)
	}
}

// GetTickerChan 获取K线数据通道
func (subscriber *QuotationSubscriber) GetTickerChan() <-chan Ticker {
	if subscriber.tickerChan == nil {
		subscriber.tickerChan = make(chan Ticker, 1024)
	}
	return subscriber.tickerChan
}

// Subscribe 订阅代币K线
// tokenAddresses: 要订阅的代币地址列表
func (subscriber *QuotationSubscriber) Subscribe(tokenAddresses []string) error {
	// 转换为小写
	for idx, tokenAddress := range tokenAddresses {
		tokenAddresses[idx] = strings.ToLower(tokenAddress)
	}

	// 存储到订阅列表
	for _, tokenAddress := range tokenAddresses {
		subscriber.tokenAddresses.Store(tokenAddress, true)
	}

	// 获取所有已订阅的代币
	allTokenAddresses := make([]string, 0)
	subscriber.tokenAddresses.Range(func(k any, v any) bool {
		allTokenAddresses = append(allTokenAddresses, k.(string))
		return true
	})

	// 发送订阅请求
	if subscriber.conn != nil {
		err := subscriber.sendSubscribe(allTokenAddresses)
		if err != nil {
			return err
		}
	}

	return nil
}

// Unsubscribe 取消订阅代币K线
// tokenAddresses: 要取消订阅的代币地址列表
func (subscriber *QuotationSubscriber) Unsubscribe(tokenAddresses []string) error {
	// 转换为小写
	for idx, tokenAddress := range tokenAddresses {
		tokenAddresses[idx] = strings.ToLower(tokenAddress)
	}

	// 从订阅列表中删除
	for _, tokenAddress := range tokenAddresses {
		subscriber.tokenAddresses.Delete(tokenAddress)
	}

	// 获取剩余已订阅的代币
	allTokenAddresses := make([]string, 0)
	subscriber.tokenAddresses.Range(func(k any, v any) bool {
		allTokenAddresses = append(allTokenAddresses, k.(string))
		return true
	})

	// 发送订阅请求更新列表
	if subscriber.conn != nil {
		err := subscriber.sendSubscribe(allTokenAddresses)
		if err != nil {
			return err
		}
	}

	return nil
}

// sendSubscribe 发送订阅请求
func (subscriber *QuotationSubscriber) sendSubscribe(tokenAddresses []string) error {
	if subscriber.conn == nil {
		return fmt.Errorf("[QuotationSubscriber] 连接未建立")
	}

	if len(tokenAddresses) == 0 {
		return nil
	}

	logger.Debugf("[QuotationSubscriber] 订阅代币K线, %+v", tokenAddresses)

	// 构建订阅数据
	data := make([]map[string]any, 0)
	for _, tokenAddress := range tokenAddresses {
		data = append(data, map[string]any{
			"chain":     subscriber.chain,
			"addresses": tokenAddress,
			"interval":  subscriber.resolution,
		})
	}

	payload := map[string]any{
		"action":  "subscribe",
		"id":      uuid.NewString(),
		"channel": "kline",
		"data":    data,
	}

	return subscriber.conn.WriteJSON(payload)
}

// run 订阅者主循环
func (subscriber *QuotationSubscriber) run() {
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
				logger.Infof("[QuotationSubscriber] 重新建立连接...")
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
func (subscriber *QuotationSubscriber) connect() {
	headers := make(http.Header)
	headers.Set("origin", "https://gmgn.ai")
	headers.Set("user-agent", utils.RandomUserAgent())
	headers.Set("accept-language", "zh-CN,zh;q=0.9")
	headers.Set("cache-control", "no-cache")
	headers.Set("pragma", "no-cache")
	headers.Set("accept-encoding", "gzip, deflate, br, zstd")

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

	// 建立WebSocket连接
	conn, _, err := dialer.DialContext(subscriber.ctx, subscriber.url, headers)
	if err != nil {
		logger.Errorf("[QuotationSubscriber] 连接失败, %v", err)
		subscriber.scheduleReconnect()
		return
	}

	subscriber.conn = conn
	subscriber.messageCounter = make(map[string]int)
	logger.Infof("[QuotationSubscriber] 连接已建立")

	// 发送已订阅代币的订阅请求
	tokenAddresses := make([]string, 0)
	subscriber.tokenAddresses.Range(func(k any, v any) bool {
		tokenAddresses = append(tokenAddresses, k.(string))
		return true
	})
	if len(tokenAddresses) > 0 {
		if err := subscriber.sendSubscribe(tokenAddresses); err != nil {
			logger.Errorf("[QuotationSubscriber] 订阅失败, %v", err)
			conn.Close()
			subscriber.scheduleReconnect()
			return
		}
		logger.Infof("[QuotationSubscriber] 订阅代币: %v", tokenAddresses)
	}

	// 启动消息读取协程
	go subscriber.readMessages()
}

// heartbeat 心跳保活
func (subscriber *QuotationSubscriber) heartbeat(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// 发送心跳消息
			msg := fmt.Sprintf(`{"action":"heartbeat","client_ts":%d}`, time.Now().UnixMilli())
			if err := subscriber.conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				logger.Errorf("[QuotationSubscriber] 发送心跳消息失败, %v", err)
				return
			}

			// 每60秒发送一次心跳
			duration := time.Second * 60
			timer.Reset(duration)
		case <-ctx.Done():
			return
		}
	}
}

// readMessages 读取WebSocket消息
func (subscriber *QuotationSubscriber) readMessages() {
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
			logger.Errorf("[QuotationSubscriber] 读取出错, %v", err)
			subscriber.scheduleReconnect()
			return
		}

		logger.Debugf("[QuotationSubscriber] 收到新消息, %s", message)

		var msg channelMessage
		if err = json.Unmarshal(message, &msg); err != nil {
			logger.Errorf("[QuotationSubscriber] 解析消息失败, message: %s, %v", message, err)
			continue
		}

		// 处理K线数据
		if msg.Channel == "kline" {
			if subscriber.tickerChan != nil {
				var klines []klineChannelData
				if err = json.Unmarshal(msg.Data, &klines); err != nil {
					logger.Errorf("[QuotationSubscriber] 解析 kline 失败, message: %s, %v", string(msg.Data), err)
					continue
				}

				for _, kline := range klines {
					// 检查是否为该代币的首条数据
					count, ok := subscriber.messageCounter[kline.A]
					if !ok {
						count = 0
					}

					ticker := Ticker{
						Token: kline.A,
						First: count == 0,
						Ohlc: charts.Ohlc{
							Open:   kline.O,
							Close:  kline.C,
							High:   kline.H,
							Low:    kline.L,
							Time:   time.Unix(kline.T, 0),
							Volume: kline.V,
						},
					}

					subscriber.messageCounter[kline.A] = count + 1

					// 发送到K线数据通道
					select {
					case subscriber.tickerChan <- ticker:
						logger.Debugf("[QuotationSubscriber] 分发 Ticker 数据, %+v", ticker)
					default:
						logger.Warnf("[QuotationSubscriber] 分发 Ticker 数据, channel 已满. %+v", ticker)
					}
				}
			}
		}
	}
}

// scheduleReconnect 安排重连
func (subscriber *QuotationSubscriber) scheduleReconnect() {
	if subscriber.ctx.Err() == nil {
		subscriber.conn = nil
		select {
		case subscriber.reconnect <- struct{}{}:
		default:
		}
	}
}
