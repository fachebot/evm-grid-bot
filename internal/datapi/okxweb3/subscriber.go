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

const (
	reconnectInitial = 1 * time.Second
	reconnectMax     = 30 * time.Second
)

type Ticker struct {
	Token string
	First bool
	Ohlc  charts.Ohlc
}

type OkxSubscriber struct {
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}
	url      string

	chainIndex string
	resolution string
	conn       *websocket.Conn
	proxy      config.Sock5Proxy
	reconnect  chan struct{}
	mutex      sync.Mutex
	assets     map[string]struct{}

	tickerChan     chan Ticker
	messageCounter map[string]int
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

func NewOkxSubscriber(chainId int64, resolution string, proxy config.Sock5Proxy) (*OkxSubscriber, error) {
	chainIndex, ok := ChainIdToChainIndex(chainId)
	if !ok {
		return nil, errors.New("unsupported chain")
	}

	ctx, cancel := context.WithCancel(context.Background())
	subscriber := &OkxSubscriber{
		ctx:            ctx,
		cancel:         cancel,
		url:            "wss://wsdexpri.okx.com/ws/v5/ipublic",
		chainIndex:     chainIndex,
		resolution:     resolution,
		proxy:          proxy,
		reconnect:      make(chan struct{}, 1),
		assets:         make(map[string]struct{}),
		messageCounter: make(map[string]int),
	}
	return subscriber, nil
}

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

func (subscriber *OkxSubscriber) WaitUntilConnected() {
	for subscriber.conn == nil {
		time.Sleep(time.Second * 1)
	}
}

func (subscriber *OkxSubscriber) GetTickerChan() <-chan Ticker {
	if subscriber.tickerChan == nil {
		subscriber.tickerChan = make(chan Ticker, 1024)
	}
	return subscriber.tickerChan
}

func (subscriber *OkxSubscriber) Subscribe(assets []string) error {
	for idx, asset := range assets {
		assets[idx] = strings.ToLower(asset)
	}

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

func (subscriber *OkxSubscriber) Unsubscribe(assets []string) error {
	if len(assets) == 0 {
		return nil
	}
	if subscriber.conn == nil {
		return fmt.Errorf("[OkxSubscriber] 连接未建立")
	}

	for idx, asset := range assets {
		assets[idx] = strings.ToLower(asset)
	}

	logger.Debugf("[OkxSubscriber] 取消订阅Candle, assets: %+v", assets)

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

				reconnectDelay *= 2
				if reconnectDelay > reconnectMax {
					reconnectDelay = reconnectMax
				}
			}
		}
	}

	subscriber.stopChan <- struct{}{}
}

func (subscriber *OkxSubscriber) connect() {
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

	conn, _, err := dialer.Dial(subscriber.url, headers)
	if err != nil {
		logger.Errorf("[OkxSubscriber] 连接失败, %v", err)
		subscriber.scheduleReconnect()
		return
	}

	subscriber.conn = conn
	subscriber.messageCounter = make(map[string]int)
	logger.Infof("[OkxSubscriber] 连接已建立")

	assets := make([]string, 0)
	subscriber.mutex.Lock()
	for asset := range subscriber.assets {
		assets = append(assets, asset)
	}
	subscriber.mutex.Unlock()
	subscriber.Subscribe(assets)

	go subscriber.readMessages()
}

func (subscriber *OkxSubscriber) heartbeat(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			if err := subscriber.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				logger.Errorf("[OkxSubscriber] 发送心跳消息失败, %v", err)
				return
			}

			duration := time.Second * 20
			timer.Reset(duration)
		case <-ctx.Done():
			return
		}
	}
}

func (subscriber *OkxSubscriber) readMessages() {
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

		if payload.Event != "" {
			continue
		}

		channel := fmt.Sprintf("dex-token-candle%s", subscriber.resolution)
		switch payload.GetChannel() {
		case channel:
			var tokenCandles [][]decimal.Decimal
			if err = json.Unmarshal(payload.Data, &tokenCandles); err != nil {
				logger.Errorf("[JupagSubscriber] 解析Candles失败, message: %s, %v", message, err)
				continue
			}

			ohlcs := make([]charts.Ohlc, 0, len(tokenCandles))
			for _, data := range tokenCandles {
				if len(data) < 8 {
					logger.Errorf("[JupagSubscriber] Candle数据长度错误, candle: %+v", data)
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

func (subscriber *OkxSubscriber) scheduleReconnect() {
	if subscriber.ctx.Err() == nil {
		select {
		case subscriber.reconnect <- struct{}{}:
		default:
		}
	}
}
