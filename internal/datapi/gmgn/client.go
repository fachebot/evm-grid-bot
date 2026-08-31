package gmgn

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/enetx/g"
	"github.com/enetx/surf"
	"github.com/fachebot/evm-grid-bot/internal/charts"
	"github.com/fachebot/evm-grid-bot/internal/config"

	"github.com/google/uuid"
)

// GMGN API基础URL
var (
	GmgnAIBaseURL  = "https://gmgn.ai"
	fakeDeviceInfo = ""
)

// getDeviceInfo 生成设备信息参数
// 用于模拟浏览器访问时的设备指纹
func getDeviceInfo() (string, error) {
	deviceId, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	var buffer [16]byte
	if _, err = rand.Read(buffer[:]); err != nil {
		return "", err
	}

	deviceInfo := url.Values{
		"device_id": []string{deviceId.String()},
		"client_id": []string{"gmgn_web_20250820-2734-a756529"},
		"from_app":  []string{"gmgn"},
		"app_ver":   []string{"20250820-2734-a756529"},
		"tz_name":   []string{"Etc/GMT-8"},
		"tz_offset": []string{"28800"},
		"app_lang":  []string{"zh-CN"},
		"fp_did":    []string{hex.EncodeToString(buffer[:])},
		"os":        []string{"web"},
	}
	return deviceInfo.Encode(), nil
}

func init() {
	var err error
	fakeDeviceInfo, err = getDeviceInfo()
	if err != nil {
		panic(err)
	}
}

// Client GMGN API客户端
type Client struct {
	chain      string            // 链名称 (bsc, base等)
	proxy      config.Sock5Proxy // SOCKS5代理配置
	httpClient *http.Client      // HTTP客户端
}

// NewClient 创建GMGN客户端实例
// chainId: 链ID (56=BSC, 8453=Base等)
// proxy: SOCKS5代理配置
func NewClient(chainId int64, proxy config.Sock5Proxy) (*Client, error) {
	chain, ok := ChainIdToChainName(chainId)
	if !ok {
		return nil, errors.New("unsupported chain")
	}

	// 使用surf库创建支持浏览器指纹的HTTP客户端
	builder := surf.NewClient().Builder()
	if proxy.Enable {
		builder.Proxy(g.String(fmt.Sprintf("socks5://%s:%d", proxy.Host, proxy.Port)))
	}
	httpClient := builder.
		Impersonate(). // 模拟浏览器
		Chrome().      // 使用Chrome指纹
		Build().
		Unwrap().
		Std() // 转换为标准http.Client

	return &Client{
		chain:      chain,
		proxy:      proxy,
		httpClient: httpClient,
	}, nil
}

// getHeaders 获取HTTP请求头
// referer: referer来源URL
func (c *Client) getHeaders(referer string) map[string]string {
	headers := map[string]string{
		"accept":          "application/json, text/plain, */*",
		"accept-language": "zh-CN,zh;q=0.9",
		"accept-encoding": "gzip, deflate, br",
	}
	if referer != "" {
		headers["referer"] = referer
	}
	return headers
}

// doRequest 执行HTTP请求
// ctx: 上下文
// url: 请求URL
// method: HTTP方法
// bodyJson: 请求体JSON对象
// referer: referer来源
func (c *Client) doRequest(ctx context.Context, url, method string, bodyJson any, referer string) (string, error) {
	var body io.Reader
	if bodyJson != nil {
		data, err := json.Marshal(bodyJson)
		if err != nil {
			return "", err
		}
		body = bytes.NewBuffer(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return "", err
	}

	headers := c.getHeaders(referer)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("http status: %d", resp.StatusCode)
	}

	return string(respBody), nil
}

// parseGmgnResponse 解析GMGN API响应
func (c *Client) parseGmgnResponse(responseBody string) (*gmgnResponse, error) {
	var res gmgnResponse
	if err := json.Unmarshal([]byte(responseBody), &res); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if res.Code != 0 {
		return nil, fmt.Errorf("gmgn api error - code: %d, msg: %s", res.Code, res.Msg)
	}

	return &res, nil
}

// convertToOhlcData 将GMGN K线数据转换为标准格式
func (c *Client) convertToOhlcData(gmgnData []gmgnOhlc) []charts.Ohlc {
	ohlcs := make([]charts.Ohlc, 0, len(gmgnData))
	for _, item := range gmgnData {
		ohlc := charts.Ohlc{
			Open:   item.Open,
			Close:  item.Close,
			High:   item.High,
			Low:    item.Low,
			Time:   time.Unix(item.Time.IntPart()/1000, 0),
			Volume: item.Volume,
		}
		ohlcs = append(ohlcs, ohlc)
	}
	return ohlcs
}

// FetchTokenCandles 获取代币K线数据
// ctx: 上下文
// token: 代币地址
// to: 结束时间
// period: K线周期 (1h, 15m等)
// limit: 返回数量限制
func (c *Client) FetchTokenCandles(ctx context.Context, token string, to time.Time, period string, limit int) ([]charts.Ohlc, error) {
	intervalD, err := charts.ResolutionToDuration(period)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/token_candles/%s/%s?%s&resolution=%s&from=0&to=%d&limit=%d",
		GmgnAIBaseURL, c.chain, token, fakeDeviceInfo, period, to.UnixMilli(), limit)
	referer := fmt.Sprintf("%s/%s/token/%s", GmgnAIBaseURL, c.chain, token)

	response, err := c.doRequest(ctx, url, http.MethodGet, nil, referer)
	if err != nil {
		return nil, err
	}

	gmgnResp, err := c.parseGmgnResponse(response)
	if err != nil {
		return nil, err
	}

	var tokenCandles gmgnTokenCandles
	if err := json.Unmarshal(gmgnResp.Data, &tokenCandles); err != nil {
		return nil, fmt.Errorf("failed to parse ohlc data: %w", err)
	}

	result := c.convertToOhlcData(tokenCandles.List)
	result = charts.FillMissingOhlc(result, to, intervalD)

	return result, nil
}

// FetchTokenHolders 获取代币持有者列表
// ctx: 上下文
// token: 代币地址
func (c *Client) FetchTokenHolders(ctx context.Context, token string) ([]*HolderInfo, error) {
	url := fmt.Sprintf("%s/vas/api/v1/token_holders/%s/%s?%s&limit=100&cost=20orderby=amount_percentage&direction=desc",
		GmgnAIBaseURL, c.chain, token, fakeDeviceInfo)
	referer := fmt.Sprintf("%s/%s/token/%s", GmgnAIBaseURL, c.chain, token)

	response, err := c.doRequest(ctx, url, http.MethodGet, nil, referer)
	if err != nil {
		return nil, err
	}

	gmgnResp, err := c.parseGmgnResponse(response)
	if err != nil {
		return nil, err
	}

	var holders HolderInfoList
	if err := json.Unmarshal(gmgnResp.Data, &holders); err != nil {
		return nil, fmt.Errorf("failed to parse holder list: %w", err)
	}

	return holders.List, nil
}

// FetchWalletHoldings 获取钱包持仓列表
// ctx: 上下文
// wallet: 钱包地址
func (c *Client) FetchWalletHoldings(ctx context.Context, wallet string) ([]*WalletHolding, error) {
	url := fmt.Sprintf("%s/api/v1/wallet_holdings/%s/%s?%s&limit=50&orderby=last_active_timestamp&direction=desc&showsmall=false&sellout=false&hide_airdrop=true&hide_abnormal=false",
		GmgnAIBaseURL, c.chain, wallet, fakeDeviceInfo)
	referer := fmt.Sprintf("%s/%s/address/%s", c.chain, GmgnAIBaseURL, wallet)

	response, err := c.doRequest(ctx, url, http.MethodGet, nil, referer)
	if err != nil {
		return nil, err
	}

	gmgnResp, err := c.parseGmgnResponse(response)
	if err != nil {
		return nil, err
	}

	var holdings WalletHoldings
	if err := json.Unmarshal(gmgnResp.Data, &holdings); err != nil {
		return nil, fmt.Errorf("failed to parse wallet holder list: %w", err)
	}

	return holdings.Holdings, nil
}

// FetchTrendingToken1H 获取1小时内热门代币
// ctx: 上下文
// tokenFilter: 代币过滤条件
func (c *Client) FetchTrendingToken1H(ctx context.Context, tokenFilter TokenFilter) (*TrendingTokens, error) {
	params := []string{
		"orderby=renowned_count",
		"direction=desc",
		"filters[]=frozen",
		"filters[]=burn",
		"filters[]=distribed",
		"platforms[]=pump",
		"platforms[]=pumpamm",
		"platforms[]=moonshot",
		"platforms[]=raydium",
		"platforms[]=meteora",
		"platforms[]=fluxbeam",
		"platforms[]=orca",
		"platforms[]=ray_launchpad",
		"platforms[]=boop",
		"platforms[]=letsbonk",
		fmt.Sprintf("min_created=%dm", tokenFilter.MinCreatedMinutes),
		fmt.Sprintf("max_created=%dm", tokenFilter.MaxCreatedMinutes),
		fmt.Sprintf("min_marketcap=%v", tokenFilter.MinMarketcap),
		fmt.Sprintf("max_marketcap=%v", tokenFilter.MaxMarketcap),
		fmt.Sprintf("min_holder_count=%d", tokenFilter.MinHolderCount),
		fmt.Sprintf("min_swaps=%d", tokenFilter.MinSwaps1H),
		fmt.Sprintf("min_volume=%v", tokenFilter.MinVolume1H),
	}

	referer := fmt.Sprintf("https://gmgn.ai/trend?chain=%s", c.chain)
	url := fmt.Sprintf("%s/defi/quotation/v1/rank/%s/swaps/1h?%s&%s", GmgnAIBaseURL, c.chain, fakeDeviceInfo, strings.Join(params, "&"))

	response, err := c.doRequest(ctx, url, http.MethodGet, nil, referer)
	if err != nil {
		return nil, err
	}

	gmgnResp, err := c.parseGmgnResponse(response)
	if err != nil {
		return nil, err
	}

	var trendingTokens TrendingTokens
	if err := json.Unmarshal(gmgnResp.Data, &trendingTokens); err != nil {
		return nil, fmt.Errorf("failed to parse trending token: %w", err)
	}

	return &trendingTokens, nil
}
