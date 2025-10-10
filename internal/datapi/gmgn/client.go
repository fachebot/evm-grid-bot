package gmgn

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fachebot/evm-grid-bot/internal/charts"
	"github.com/fachebot/evm-grid-bot/internal/config"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/google/uuid"
	"golang.org/x/net/proxy"
)

const (
	// API endpoints
	gmgnAIBaseURL = "https://gmgn.ai"

	// HTTP headers
	defaultJA3       = "771,4865-4867-4866-49195-49199-52393-52392-49196-49200-49162-49161-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-51-43-13-45-28-21,29-23-24-25-256-257,0"
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36"
)

var (
	fakeDeviceInfo = ""
)

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

type Client struct {
	chain      string
	proxy      config.Sock5Proxy
	httpClient cycletls.CycleTLS
	zenRows    config.ZenRows
}

func NewClient(chainId int64, proxy config.Sock5Proxy, zenRows config.ZenRows) (*Client, error) {
	chain, ok := ChainIdToChainName(chainId)
	if !ok {
		return nil, errors.New("unsupported chain")
	}

	return &Client{
		chain:      chain,
		proxy:      proxy,
		httpClient: cycletls.Init(),
		zenRows:    zenRows,
	}, nil
}

// getProxyURL 获取代理URL
func (c *Client) getProxyURL() string {
	if !c.proxy.Enable {
		return ""
	}
	return fmt.Sprintf("socks5://%s:%d", c.proxy.Host, c.proxy.Port)
}

// getCommonHeaders 获取通用请求头
func (c *Client) getCommonHeaders() map[string]string {
	return map[string]string{
		"accept":                      "application/json, text/plain, */*",
		"accept-language":             "zh-CN,zh;q=0.9",
		"accept-encoding":             "gzip,deflate,br",
		"priority":                    "u=1, i",
		"sec-ch-ua":                   `"Google Chrome";v="135", "Not-A.Brand";v="8", "Chromium";v="135"`,
		"sec-ch-ua-arch":              `"x86"`,
		"sec-ch-ua-bitness":           `"64"`,
		"sec-ch-ua-full-version":      `"136.0.7103.93"`,
		"sec-ch-ua-full-version-list": `"Chromium";v="136.0.7103.93", "Google Chrome";v="136.0.7103.93", "Not.A/Brand";v="99.0.0.0"`,
		"sec-ch-ua-mobile":            `?0`,
		"sec-ch-ua-model":             `""`,
		"sec-ch-ua-platform":          `"Windows"`,
		"sec-ch-ua-platform-version":  `"Windows"`,
		"sec-fetch-dest":              `empty`,
		"sec-fetch-mode":              `cors`,
		"sec-fetch-site":              `same-origin`,
	}
}

// getRequestOptions 获取请求选项
func (c *Client) getRequestOptions(referer string) cycletls.Options {
	headers := c.getCommonHeaders()
	if referer != "" {
		headers["referer"] = referer
	}

	return cycletls.Options{
		Proxy:     c.getProxyURL(),
		Ja3:       defaultJA3,
		UserAgent: defaultUserAgent,
		Headers:   headers,
	}
}

// doRequest 执行HTTP请求并处理响应
func (c *Client) doRequest(ctx context.Context, scraperApiKey, url, method string, bodyJson any, referer string) (string, error) {
	if scraperApiKey == "" {
		var body []byte
		if bodyJson != nil {
			var err error
			body, err = json.Marshal(bodyJson)
			if err != nil {
				return "", err
			}
		}

		options := c.getRequestOptions(referer)
		if body != nil {
			options.Body = string(body)
		}

		response, err := c.httpClient.Do(url, options, method)
		if err != nil {
			return "", fmt.Errorf("request failed: %w", err)
		}

		if response.Status < 200 || response.Status >= 300 {
			return "", fmt.Errorf("http status: %d", response.Status)
		}

		return response.Body, nil
	}

	httpClient := new(http.Client)
	if c.proxy.Enable {
		socks5Proxy := fmt.Sprintf("%s:%d", c.proxy.Host, c.proxy.Port)
		dialer, err := proxy.SOCKS5("tcp", socks5Proxy, nil, proxy.Direct)
		if err != nil {
			return "", err
		}

		httpClient.Transport = &http.Transport{
			Dial:            dialer.Dial,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return scraperDoRequest(ctx, httpClient, scraperApiKey, method, url, bodyJson)
}

// parseGmgnResponse 解析GMGN响应
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

// convertToOhlcData 转换GMGN OHLC数据为标准格式
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

func (c *Client) FetchTokenCandles(ctx context.Context, token string, to time.Time, period string, limit int) ([]charts.Ohlc, error) {
	intervalD, err := charts.ResolutionToDuration(period)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/token_candles/%s/%s?%s&resolution=%s&from=0&to=%d&limit=%d",
		gmgnAIBaseURL, c.chain, token, fakeDeviceInfo, period, to.UnixMilli(), limit)
	referer := fmt.Sprintf("%s/%s/token/%s", gmgnAIBaseURL, c.chain, token)

	scraperApiKey := ""
	if c.zenRows.FetchTokenCandles {
		scraperApiKey = c.zenRows.Apikey
	}
	response, err := c.doRequest(ctx, scraperApiKey, url, http.MethodGet, nil, referer)
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

func (c *Client) FetchTokenHolders(ctx context.Context, token string) ([]*HolderInfo, error) {
	url := fmt.Sprintf("%s/vas/api/v1/token_holders/%s/%s?%s&limit=100&cost=20orderby=amount_percentage&direction=desc",
		gmgnAIBaseURL, c.chain, token, fakeDeviceInfo)
	referer := fmt.Sprintf("%s/%s/token/%s", gmgnAIBaseURL, c.chain, token)

	scraperApiKey := ""
	if c.zenRows.FetchTokenHolders {
		scraperApiKey = c.zenRows.Apikey
	}
	response, err := c.doRequest(ctx, scraperApiKey, url, http.MethodGet, nil, referer)
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

func (c *Client) FetchWalletHoldings(ctx context.Context, wallet string) ([]*WalletHolding, error) {
	url := fmt.Sprintf("%s/api/v1/wallet_holdings/%s/%s?%s&limit=50&orderby=last_active_timestamp&direction=desc&showsmall=false&sellout=false&hide_airdrop=true&hide_abnormal=false",
		gmgnAIBaseURL, c.chain, wallet, fakeDeviceInfo)
	referer := fmt.Sprintf("%s/%s/address/%s", c.chain, gmgnAIBaseURL, wallet)

	scraperApiKey := ""
	if c.zenRows.FetchWalletHoldings {
		scraperApiKey = c.zenRows.Apikey
	}
	response, err := c.doRequest(ctx, scraperApiKey, url, http.MethodGet, nil, referer)
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
	url := fmt.Sprintf("%s/defi/quotation/v1/rank/%s/swaps/1h?%s&%s", gmgnAIBaseURL, c.chain, fakeDeviceInfo, strings.Join(params, "&"))

	scraperApiKey := ""
	if c.zenRows.FetchWalletHoldings {
		scraperApiKey = c.zenRows.Apikey
	}
	response, err := c.doRequest(ctx, scraperApiKey, url, http.MethodGet, nil, referer)
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
