package gmgn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fachebot/evm-grid-bot/internal/config"
	"github.com/shopspring/decimal"
)

func TestClient_parseGmgnResponse(t *testing.T) {
	client := &Client{}

	t.Run("success response", func(t *testing.T) {
		resp := `{"code":0,"msg":"success","data":"{}"}`
		result, err := client.parseGmgnResponse(resp)
		if err != nil {
			t.Fatalf("parseGmgnResponse failed: %v", err)
		}
		if result.Code != 0 {
			t.Errorf("expected code 0, got %d", result.Code)
		}
	})

	t.Run("error response", func(t *testing.T) {
		resp := `{"code":1,"msg":"error","data":""}`
		_, err := client.parseGmgnResponse(resp)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		resp := `invalid json`
		_, err := client.parseGmgnResponse(resp)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty data", func(t *testing.T) {
		resp := `{"code":0,"msg":"success","data":null}`
		result, err := client.parseGmgnResponse(resp)
		if err != nil {
			t.Fatalf("parseGmgnResponse failed: %v", err)
		}
		if result.Data == nil {
			t.Error("expected data to not be nil")
		}
	})
}

func TestClient_getHeaders(t *testing.T) {
	client := &Client{}

	t.Run("without referer", func(t *testing.T) {
		headers := client.getHeaders("")
		if headers["accept"] == "" {
			t.Error("accept should be set")
		}
		if headers["accept-language"] == "" {
			t.Error("accept-language should be set")
		}
		if headers["accept-encoding"] == "" {
			t.Error("accept-encoding should be set")
		}
		if headers["referer"] != "" {
			t.Error("referer should not be set when empty")
		}
	})

	t.Run("with referer", func(t *testing.T) {
		headers := client.getHeaders("https://example.com")
		if headers["referer"] != "https://example.com" {
			t.Errorf("expected referer https://example.com, got %s", headers["referer"])
		}
	})
}

func TestClient_convertToOhlcData(t *testing.T) {
	client := &Client{}

	t.Run("empty input", func(t *testing.T) {
		result := client.convertToOhlcData([]gmgnOhlc{})
		if len(result) != 0 {
			t.Errorf("expected 0 ohlc, got %d", len(result))
		}
	})

	t.Run("single candle", func(t *testing.T) {
		input := []gmgnOhlc{
			{
				Open:   decimal.NewFromFloat(1.0),
				Close:  decimal.NewFromFloat(2.0),
				High:   decimal.NewFromFloat(2.5),
				Low:    decimal.NewFromFloat(0.5),
				Time:   decimal.NewFromInt(1700000000000),
				Volume: decimal.NewFromFloat(1000),
			},
		}
		result := client.convertToOhlcData(input)
		if len(result) != 1 {
			t.Fatalf("expected 1 ohlc, got %d", len(result))
		}
		if !result[0].Open.Equal(decimal.NewFromFloat(1.0)) {
			t.Errorf("expected open 1.0, got %v", result[0].Open)
		}
		if !result[0].Close.Equal(decimal.NewFromFloat(2.0)) {
			t.Errorf("expected close 2.0, got %v", result[0].Close)
		}
		if !result[0].High.Equal(decimal.NewFromFloat(2.5)) {
			t.Errorf("expected high 2.5, got %v", result[0].High)
		}
		if !result[0].Low.Equal(decimal.NewFromFloat(0.5)) {
			t.Errorf("expected low 0.5, got %v", result[0].Low)
		}
		if !result[0].Volume.Equal(decimal.NewFromFloat(1000)) {
			t.Errorf("expected volume 1000, got %v", result[0].Volume)
		}
	})

	t.Run("multiple candles", func(t *testing.T) {
		input := []gmgnOhlc{
			{Open: decimal.NewFromFloat(1.0), Close: decimal.NewFromFloat(1.5), High: decimal.NewFromFloat(2.0), Low: decimal.NewFromFloat(0.5), Time: decimal.NewFromInt(1700000000000), Volume: decimal.NewFromFloat(100)},
			{Open: decimal.NewFromFloat(1.5), Close: decimal.NewFromFloat(2.0), High: decimal.NewFromFloat(2.5), Low: decimal.NewFromFloat(1.0), Time: decimal.NewFromInt(1700000060000), Volume: decimal.NewFromFloat(200)},
			{Open: decimal.NewFromFloat(2.0), Close: decimal.NewFromFloat(1.8), High: decimal.NewFromFloat(2.2), Low: decimal.NewFromFloat(1.5), Time: decimal.NewFromInt(1700000120000), Volume: decimal.NewFromFloat(150)},
		}
		result := client.convertToOhlcData(input)
		if len(result) != 3 {
			t.Fatalf("expected 3 ohlc, got %d", len(result))
		}
	})
}

func TestNewClient(t *testing.T) {
	t.Run("unsupported chain", func(t *testing.T) {
		_, err := NewClient(99999, config.Sock5Proxy{})
		if err == nil {
			t.Fatal("expected error for unsupported chain")
		}
	})

	t.Run("valid chain", func(t *testing.T) {
		client, err := NewClient(56, config.Sock5Proxy{})
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		if client == nil {
			t.Fatal("client should not be nil")
		}
		if client.httpClient == nil {
			t.Error("httpClient should not be nil")
		}
	})

	t.Run("with proxy enabled", func(t *testing.T) {
		proxy := config.Sock5Proxy{
			Host:   "127.0.0.1",
			Port:   1080,
			Enable: true,
		}
		client, err := NewClient(56, proxy)
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		if client == nil {
			t.Fatal("client should not be nil")
		}
	})
}

func TestClient_FetchTokenCandles(t *testing.T) {
	originalBaseURL := GmgnAIBaseURL
	GmgnAIBaseURL = "http://test-server"
	defer func() { GmgnAIBaseURL = originalBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/token_candles/bsc/0x123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("resolution") != "1h" {
			t.Errorf("unexpected resolution: %s", r.URL.Query().Get("resolution"))
		}
		if r.URL.Query().Get("limit") != "100" {
			t.Errorf("unexpected limit: %s", r.URL.Query().Get("limit"))
		}

		resp := gmgnResponse{
			Code: 0,
			Msg:  "success",
			Data: json.RawMessage(`{"list":[{"open":"0.001","close":"0.002","high":"0.003","low":"0.001","time":"1700000000000","volume":"1000"}]}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	GmgnAIBaseURL = server.URL

	client := &Client{
		chain:      "bsc",
		httpClient: server.Client(),
	}

	to := time.Unix(1700000000, 0).Add(time.Hour)
	result, err := client.FetchTokenCandles(context.Background(), "0x123", to, "1h", 100)
	if err != nil {
		t.Fatalf("FetchTokenCandles failed: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("expected at least 1 candle")
	}

	firstCandle := result[0]
	if !firstCandle.Open.Equal(decimal.NewFromFloat(0.001)) {
		t.Errorf("expected open 0.001, got %v", firstCandle.Open)
	}
	if !firstCandle.Close.Equal(decimal.NewFromFloat(0.002)) {
		t.Errorf("expected close 0.002, got %v", firstCandle.Close)
	}
	if !firstCandle.High.Equal(decimal.NewFromFloat(0.003)) {
		t.Errorf("expected high 0.003, got %v", firstCandle.High)
	}
	if !firstCandle.Low.Equal(decimal.NewFromFloat(0.001)) {
		t.Errorf("expected low 0.001, got %v", firstCandle.Low)
	}
	if !firstCandle.Volume.Equal(decimal.NewFromFloat(1000)) {
		t.Errorf("expected volume 1000, got %v", firstCandle.Volume)
	}
}

func TestClient_FetchTokenCandles_APIError(t *testing.T) {
	originalBaseURL := GmgnAIBaseURL
	GmgnAIBaseURL = "http://test-server"
	defer func() { GmgnAIBaseURL = originalBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := gmgnResponse{
			Code: 1,
			Msg:  "error message",
			Data: json.RawMessage(`{}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	GmgnAIBaseURL = server.URL

	client := &Client{
		chain:      "bsc",
		httpClient: server.Client(),
	}

	to := time.Now()
	_, err := client.FetchTokenCandles(context.Background(), "0x123", to, "1h", 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_FetchTokenCandles_InvalidJSON(t *testing.T) {
	originalBaseURL := GmgnAIBaseURL
	GmgnAIBaseURL = "http://test-server"
	defer func() { GmgnAIBaseURL = originalBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	GmgnAIBaseURL = server.URL

	client := &Client{
		chain:      "bsc",
		httpClient: server.Client(),
	}

	to := time.Now()
	_, err := client.FetchTokenCandles(context.Background(), "0x123", to, "1h", 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_FetchTokenCandles_HttpError(t *testing.T) {
	originalBaseURL := GmgnAIBaseURL
	GmgnAIBaseURL = "http://test-server"
	defer func() { GmgnAIBaseURL = originalBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	GmgnAIBaseURL = server.URL

	client := &Client{
		chain:      "bsc",
		httpClient: server.Client(),
	}

	to := time.Now()
	_, err := client.FetchTokenCandles(context.Background(), "0x123", to, "1h", 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_FetchTokenCandles_InvalidOhlcData(t *testing.T) {
	originalBaseURL := GmgnAIBaseURL
	GmgnAIBaseURL = "http://test-server"
	defer func() { GmgnAIBaseURL = originalBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := gmgnResponse{
			Code: 0,
			Msg:  "success",
			Data: json.RawMessage(`{"list":[{"open":"not-a-number","close":"0.002","high":"0.003","low":"0.001","time":"1700000000000","volume":"1000"}]}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	GmgnAIBaseURL = server.URL

	client := &Client{
		chain:      "bsc",
		httpClient: server.Client(),
	}

	to := time.Unix(1700000000, 0).Add(time.Hour)
	_, err := client.FetchTokenCandles(context.Background(), "0x123", to, "1h", 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
