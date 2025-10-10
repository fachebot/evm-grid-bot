package okxweb3

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

type Client struct {
	apiKey     string
	secretKey  []byte
	passphrase string
	client     *http.Client
}

func NewClient(apiKey, secretKey, passphrase string, transportProxy *http.Transport) *Client {
	httpClient := new(http.Client)
	if transportProxy != nil {
		httpClient.Transport = transportProxy
	}

	return &Client{
		apiKey:     apiKey,
		secretKey:  []byte(secretKey),
		passphrase: passphrase,
		client:     httpClient,
	}
}

func (client *Client) GetSupportedChains(ctx context.Context) ([]ChainInfo, error) {
	res, err := client.request(ctx, http.MethodGet, "/api/v5/wallet/chain/supported-chains", true, nil, nil)
	if err != nil {
		return nil, err
	}

	var chains []ChainInfo
	if err = client.toJSON(res, &chains); err != nil {
		return nil, err
	}
	return chains, nil
}

func (client *Client) GetRealtimePrice(ctx context.Context, chainIndex string, tokenAddresses []string) (map[string]decimal.Decimal, error) {
	if len(tokenAddresses) == 0 {
		return nil, nil
	}

	params := []map[string]string{}
	for _, tokenAddress := range tokenAddresses {
		params = append(params, map[string]string{
			"chainIndex":   chainIndex,
			"tokenAddress": tokenAddress,
		})
	}
	res, err := client.request(ctx, http.MethodPost, "/api/v5/wallet/token/real-time-price", true, nil, params)
	if err != nil {
		return nil, err
	}

	var realtimePrices []RealtimePrice
	if err = client.toJSON(res, &realtimePrices); err != nil {
		return nil, err
	}

	result := lo.SliceToMap(realtimePrices, func(item RealtimePrice) (string, decimal.Decimal) {
		return item.TokenAddress, item.Price
	})
	return result, nil
}

func (client *Client) GetAllTokenBalancesByAddress(ctx context.Context, chainIndex string, address string) ([]TokenBalance, error) {
	if chainIndex == "" || address == "" {
		return nil, errors.New("chainIndex and address cannot be empty")
	}

	params := map[string]string{
		"address":          address,
		"chains":           chainIndex,
		"excludeRiskToken": "0",
	}
	res, err := client.request(ctx, http.MethodGet, "/api/v5/dex/balance/all-token-balances-by-address", true, params, nil)
	if err != nil {
		return nil, err
	}

	var tokenAssets []TokenAssets
	if err = client.toJSON(res, &tokenAssets); err != nil {
		return nil, err
	}

	if len(tokenAssets) == 0 {
		return nil, nil
	}
	return tokenAssets[0].TokenAssets, nil
}

func (client *Client) sign(method, path, body string) (string, string) {
	format := "2006-01-02T15:04:05.999Z07:00"
	t := time.Now().UTC().Format(format)
	ts := fmt.Sprint(t)
	s := ts + method + path + body
	p := []byte(s)
	h := hmac.New(sha256.New, client.secretKey)
	h.Write(p)
	return ts, base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (client *Client) toJSON(res *http.Response, v interface{}) error {
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.New("failed to read response body: " + err.Error())
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("error response: %s", string(resBody))
	}

	var resData restResponse
	if err = json.Unmarshal(resBody, &resData); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resData.Code != "0" {
		return fmt.Errorf("error code %s: %s", resData.Code, resData.Msg)
	}

	if err = json.Unmarshal(resData.Data, v); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

func (client *Client) request(ctx context.Context, method, path string, private bool, urlParams map[string]string, jsonParams any) (*http.Response, error) {
	u := fmt.Sprintf("https://web3.okx.com%s", path)
	var (
		r    *http.Request
		err  error
		j    []byte
		body string
	)
	if method == http.MethodGet {
		r, err = http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}

		if len(urlParams) > 0 {
			q := r.URL.Query()
			for k, v := range urlParams {
				q.Add(k, strings.ReplaceAll(v, "\"", ""))
			}
			r.URL.RawQuery = q.Encode()
			if len(urlParams) > 0 {
				path += "?" + r.URL.RawQuery
			}
		}
	} else {
		j, err = json.Marshal(jsonParams)
		if err != nil {
			return nil, err
		}
		body = string(j)
		if body == "{}" {
			body = ""
		}
		r, err = http.NewRequestWithContext(ctx, method, u, bytes.NewBuffer(j))
		if err != nil {
			return nil, err
		}
		r.Header.Add("Content-Type", "application/json")
	}
	if err != nil {
		return nil, err
	}
	if private {
		timestamp, sign := client.sign(method, path, body)
		r.Header.Add("OK-ACCESS-KEY", client.apiKey)
		r.Header.Add("OK-ACCESS-PASSPHRASE", client.passphrase)
		r.Header.Add("OK-ACCESS-SIGN", sign)
		r.Header.Add("OK-ACCESS-TIMESTAMP", timestamp)
	}

	return client.client.Do(r)
}
