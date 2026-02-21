package okxweb3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/enetx/g"
	"github.com/enetx/surf"
	"github.com/fachebot/evm-grid-bot/internal/charts"
	"github.com/fachebot/evm-grid-bot/internal/config"

	"github.com/shopspring/decimal"
)

var OkxBaseURL = "https://web3.okx.com"

type Client struct {
	chainIndex string
	proxy      config.Sock5Proxy
	httpClient *http.Client
}

func NewClient(chainId int64, proxy config.Sock5Proxy) (*Client, error) {
	chainIndex, ok := ChainIdToChainIndex(chainId)
	if !ok {
		return nil, errors.New("unsupported chain")
	}

	surfClient := surf.NewClient()
	if proxy.Enable {
		surfClient.Builder().Proxy(g.String(fmt.Sprintf("socks5://%s:%d", proxy.Host, proxy.Port)))
	}
	httpClient := surfClient.Builder().
		Impersonate().
		Chrome().
		Build().
		Unwrap().
		Std()

	return &Client{
		chainIndex: chainIndex,
		proxy:      proxy,
		httpClient: httpClient,
	}, nil
}

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
		OkxBaseURL, c.chainIndex, token, to.UnixMilli(), interval, limit, time.Now().Unix())

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
