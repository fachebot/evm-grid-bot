package okxweb3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/fachebot/evm-grid-bot/internal/charts"
	"github.com/fachebot/evm-grid-bot/internal/config"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/shopspring/decimal"
)

const (
	// API endpoints
	okxBaseURL = "https://web3.okx.com"

	// HTTP headers
	defaultJA3       = "771,4865-4867-4866-49195-49199-52393-52392-49196-49200-49162-49161-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-51-43-13-45-28-21,29-23-24-25-256-257,0"
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36"
)

type Client struct {
	chainIndex string
	proxy      config.Sock5Proxy
	httpClient cycletls.CycleTLS
}

func NewClient(chainId int64, proxy config.Sock5Proxy) (*Client, error) {
	chainIndex, ok := ChainIdToChainIndex(chainId)
	if !ok {
		return nil, errors.New("unsupported chain")
	}

	return &Client{
		chainIndex: chainIndex,
		proxy:      proxy,
		httpClient: cycletls.Init(),
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
func (c *Client) doRequest(ctx context.Context, url, method string, bodyJson any, referer string) (string, error) {
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

// parseGmgnResponse 解析OKX响应
func (c *Client) parseOkxResponse(responseBody string) (*okxResponse, error) {
	var res okxResponse
	if err := json.Unmarshal([]byte(responseBody), &res); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if res.Code != "0" {
		return nil, fmt.Errorf("okx api error - code: %s, msg: %s", res.Code, res.Msg)
	}

	return &res, nil
}

func (c *Client) FetchTokenCandles(ctx context.Context, token string, to time.Time, interval string, limit int) ([]charts.Ohlc, error) {
	intervalD, err := charts.ResolutionToDuration(interval)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/priapi/v5/dex/token/market/dex-token-hlc-candles?chainId=%s&address=%s&after=%d&bar=%s&limit=%d&t=%d",
		okxBaseURL, c.chainIndex, token, to.UnixMilli(), interval, limit, time.Now().Unix())

	response, err := c.doRequest(ctx, url, http.MethodGet, nil, "https://web3.okx.com/")
	if err != nil {
		return nil, err
	}

	okxResp, err := c.parseOkxResponse(response)
	if err != nil {
		return nil, err
	}

	var tokenCandles [][]decimal.Decimal
	if err := json.Unmarshal(okxResp.Data, &tokenCandles); err != nil {
		return nil, fmt.Errorf("failed to parse ohlc data: %w", err)
	}

	result := make([]charts.Ohlc, 0, len(tokenCandles))
	for _, data := range tokenCandles {
		if len(data) < 8 {
			return nil, fmt.Errorf("failed to parse ohlc data: %+v", data)
		}

		result = append(result, charts.Ohlc{
			Open:   data[1],
			Close:  data[2],
			High:   data[3],
			Low:    data[4],
			Time:   time.UnixMilli(data[0].IntPart()),
			Volume: data[6],
		})
	}

	slices.Reverse(result)
	result = charts.FillMissingOhlc(result, to, intervalD)

	return result, nil
}
