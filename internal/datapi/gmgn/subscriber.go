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

const (
	reconnectInitial = 1 * time.Second
	reconnectMax     = 30 * time.Second
)

type Ticker struct {
	Token string
	First bool
	Ohlc  charts.Ohlc
}

type channelMessage struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}

type klineChannelData struct {
	N string          `json:"n"`
	A string          `json:"a"`
	I string          `json:"i"`
	O decimal.Decimal `json:"o"`
	H decimal.Decimal `json:"h"`
	L decimal.Decimal `json:"l"`
	C decimal.Decimal `json:"c"`
	V decimal.Decimal `json:"v"`
	T int64           `json:"t"`
}

type QuotationSubscriber struct {
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}

	conn *websocket.Conn
	url  string

	chain          string
	resolution     string
	tokenAddresses sync.Map
	proxy          config.Sock5Proxy
	reconnect      chan struct{}

	tickerChan     chan Ticker
	messageCounter map[string]int
}

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
		url:            "wss://ws.gmgn.ai/quotation",
		chain:          chain,
		resolution:     resolution,
		proxy:          proxy,
		reconnect:      make(chan struct{}, 1),
		ctx:            ctx,
		cancel:         cancel,
		messageCounter: make(map[string]int),
	}

	for _, tokenAddress := range tokenAddresses {
		subscriber.tokenAddresses.Store(tokenAddress, true)
	}
	return subscriber, nil
}

func netDialTLSContext(ctx context.Context, network, addr string, sock5Proxy string) (net.Conn, error) {
	serverName := addr
	if host, _, err := net.SplitHostPort(addr); err == nil {
		serverName = host
	}

	spec, err := utls.UTLSIdToSpec(RandomClientHelloID())
	if err != nil {
		return nil, err
	}
	for _, ext := range spec.Extensions {
		alpnExt, ok := ext.(*utls.ALPNExtension)
		if !ok {
			continue
		}

		alpnExt.AlpnProtocols = []string{"http/1.1"}
	}

	var conn net.Conn
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

	config := &utls.Config{
		InsecureSkipVerify: true,
		ServerName:         serverName,
	}

	client := utls.UClient(conn, config, utls.HelloCustom)
	if err = client.ApplyPreset(&spec); err != nil {
		return nil, err
	}

	return client, nil
}

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

func (subscriber *QuotationSubscriber) WaitUntilConnected() {
	for subscriber.conn == nil {
		time.Sleep(time.Second * 1)
	}
}

func (subscriber *QuotationSubscriber) GetTickerChan() <-chan Ticker {
	if subscriber.tickerChan == nil {
		subscriber.tickerChan = make(chan Ticker, 1024)
	}
	return subscriber.tickerChan
}

func (subscriber *QuotationSubscriber) Subscribe(tokenAddresses []string) error {
	for idx, tokenAddress := range tokenAddresses {
		tokenAddresses[idx] = strings.ToLower(tokenAddress)
	}

	for _, tokenAddress := range tokenAddresses {
		subscriber.tokenAddresses.Store(tokenAddress, true)
	}

	allTokenAddresses := make([]string, 0)
	subscriber.tokenAddresses.Range(func(k any, v any) bool {
		allTokenAddresses = append(allTokenAddresses, k.(string))
		return true
	})

	if subscriber.conn != nil {
		err := subscriber.sendSubscribe(allTokenAddresses)
		if err != nil {
			return err
		}
	}

	return nil
}

func (subscriber *QuotationSubscriber) Unsubscribe(tokenAddresses []string) error {
	for idx, tokenAddress := range tokenAddresses {
		tokenAddresses[idx] = strings.ToLower(tokenAddress)
	}

	for _, tokenAddress := range tokenAddresses {
		subscriber.tokenAddresses.Delete(tokenAddress)
	}

	allTokenAddresses := make([]string, 0)
	subscriber.tokenAddresses.Range(func(k any, v any) bool {
		allTokenAddresses = append(allTokenAddresses, k.(string))
		return true
	})

	if subscriber.conn != nil {
		err := subscriber.sendSubscribe(allTokenAddresses)
		if err != nil {
			return err
		}
	}

	return nil
}

func (subscriber *QuotationSubscriber) sendSubscribe(tokenAddresses []string) error {
	if subscriber.conn == nil {
		return fmt.Errorf("[QuotationSubscriber] 连接未建立")
	}

	if len(tokenAddresses) == 0 {
		return nil
	}

	logger.Debugf("[QuotationSubscriber] 订阅代币K线, %+v", tokenAddresses)

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

				reconnectDelay *= 2
				if reconnectDelay > reconnectMax {
					reconnectDelay = reconnectMax
				}
			}
		}
	}

	subscriber.stopChan <- struct{}{}
}

func (subscriber *QuotationSubscriber) connect() {
	headers := make(http.Header)
	headers.Set("origin", "https://gmgn.ai")
	headers.Set("user-agent", utils.RandomUserAgent())
	headers.Set("accept-language", "zh-CN,zh;q=0.9")
	headers.Set("cache-control", "no-cache")
	headers.Set("pragma", "no-cache")
	headers.Set("accept-encoding", "gzip, deflate, br, zstd")

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

	conn, _, err := dialer.DialContext(subscriber.ctx, subscriber.url, headers)
	if err != nil {
		logger.Errorf("[QuotationSubscriber] 连接失败, %v", err)
		subscriber.scheduleReconnect()
		return
	}

	subscriber.conn = conn
	subscriber.messageCounter = make(map[string]int)
	logger.Infof("[QuotationSubscriber] 连接已建立")

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

	go subscriber.readMessages()
}

func (subscriber *QuotationSubscriber) heartbeat(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			msg := fmt.Sprintf(`{"action":"heartbeat","client_ts":%d}`, time.Now().UnixMilli())
			if err := subscriber.conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				logger.Errorf("[QuotationSubscriber] 发送心跳消息失败, %v", err)
				return
			}

			duration := time.Second * 60
			timer.Reset(duration)
		case <-ctx.Done():
			return
		}
	}
}

func (subscriber *QuotationSubscriber) readMessages() {
	defer subscriber.conn.Close()

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

		if msg.Channel == "kline" {
			if subscriber.tickerChan != nil {
				var klines []klineChannelData
				if err = json.Unmarshal(msg.Data, &klines); err != nil {
					logger.Errorf("[QuotationSubscriber] 解析 kline 失败, message: %s, %v", string(msg.Data), err)
					continue
				}

				for _, kline := range klines {
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

func (subscriber *QuotationSubscriber) scheduleReconnect() {
	if subscriber.ctx.Err() == nil {
		subscriber.conn = nil
		select {
		case subscriber.reconnect <- struct{}{}:
		default:
		}
	}
}
