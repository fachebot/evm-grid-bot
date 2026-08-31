package okxweb3

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

func TestClient_parseOkxResponse(t *testing.T) {
	client := &Client{}

	t.Run("success response", func(t *testing.T) {
		resp := `{"code":"0","msg":"success","data":"{}"}`
		result, err := client.parseOkxResponse(resp)
		if err != nil {
			t.Fatalf("parseOkxResponse failed: %v", err)
		}
		if result.Code != "0" {
			t.Errorf("expected code 0, got %s", result.Code)
		}
	})

	t.Run("error response", func(t *testing.T) {
		resp := `{"code":"1","msg":"error","data":""}`
		_, err := client.parseOkxResponse(resp)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		resp := `invalid json`
		_, err := client.parseOkxResponse(resp)
		if err == nil {
			t.Fatal("expected error, got nil")
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

func TestNewClient(t *testing.T) {
	t.Run("unsupported chain", func(t *testing.T) {
		_, err := NewClient(99999, config.Sock5Proxy{})
		if err == nil {
			t.Fatal("expected error for unsupported chain")
		}
	})

	t.Run("valid chain BSC", func(t *testing.T) {
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
		if client.chainIndex != "56" {
			t.Errorf("expected chainIndex 56, got %s", client.chainIndex)
		}
	})

	t.Run("valid chain Base", func(t *testing.T) {
		client, err := NewClient(8453, config.Sock5Proxy{})
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		if client == nil {
			t.Fatal("client should not be nil")
		}
		if client.chainIndex != "8453" {
			t.Errorf("expected chainIndex 8453, got %s", client.chainIndex)
		}
	})

	t.Run("valid chain Robinhood", func(t *testing.T) {
		client, err := NewClient(4663, config.Sock5Proxy{})
		if err != nil {
			t.Fatalf("NewClient failed: %v", err)
		}
		if client == nil {
			t.Fatal("client should not be nil")
		}
		if client.chainIndex != "4663" {
			t.Errorf("expected chainIndex 4663, got %s", client.chainIndex)
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

func TestChainIdToChainIndex(t *testing.T) {
	tests := []struct {
		chainId    int64
		expected   string
		shouldFail bool
	}{
		{56, "56", false},
		{8453, "8453", false},
		{4663, "4663", false},
		{99999, "", true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result, ok := ChainIdToChainIndex(tt.chainId)
			if tt.shouldFail {
				if ok {
					t.Error("expected failure, got success")
				}
			} else {
				if !ok {
					t.Error("expected success, got failure")
				}
				if result != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestClient_FetchTokenCandles(t *testing.T) {
	originalBaseURL := OkxBaseURL
	OkxBaseURL = "http://test-server"
	defer func() { OkxBaseURL = originalBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/priapi/v5/dex/token/market/dex-token-hlc-candles" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("chainId") != "56" {
			t.Errorf("unexpected chainId: %s", r.URL.Query().Get("chainId"))
		}
		if r.URL.Query().Get("bar") != "1h" {
			t.Errorf("unexpected bar: %s", r.URL.Query().Get("bar"))
		}
		if r.URL.Query().Get("limit") != "100" {
			t.Errorf("unexpected limit: %s", r.URL.Query().Get("limit"))
		}

		resp := okxResponse{
			Code: "0",
			Msg:  "success",
			Data: json.RawMessage(`[["1700000000000","0.001","0.002","0.003","0.0005","1000","500","100"]]`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	OkxBaseURL = server.URL

	client := &Client{
		chainIndex: "56",
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
	if !firstCandle.Low.Equal(decimal.NewFromFloat(0.0005)) {
		t.Errorf("expected low 0.0005, got %v", firstCandle.Low)
	}
	if !firstCandle.Volume.Equal(decimal.NewFromFloat(500)) {
		t.Errorf("expected volume 500, got %v", firstCandle.Volume)
	}
}

func TestClient_FetchTokenCandles_APIError(t *testing.T) {
	originalBaseURL := OkxBaseURL
	OkxBaseURL = "http://test-server"
	defer func() { OkxBaseURL = originalBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := okxResponse{
			Code: "1",
			Msg:  "error message",
			Data: json.RawMessage(`{}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	OkxBaseURL = server.URL

	client := &Client{
		chainIndex: "56",
		httpClient: server.Client(),
	}

	to := time.Now()
	_, err := client.FetchTokenCandles(context.Background(), "0x123", to, "1h", 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_FetchTokenCandles_InvalidJSON(t *testing.T) {
	originalBaseURL := OkxBaseURL
	OkxBaseURL = "http://test-server"
	defer func() { OkxBaseURL = originalBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	OkxBaseURL = server.URL

	client := &Client{
		chainIndex: "56",
		httpClient: server.Client(),
	}

	to := time.Now()
	_, err := client.FetchTokenCandles(context.Background(), "0x123", to, "1h", 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_FetchTokenCandles_HttpError(t *testing.T) {
	originalBaseURL := OkxBaseURL
	OkxBaseURL = "http://test-server"
	defer func() { OkxBaseURL = originalBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	OkxBaseURL = server.URL

	client := &Client{
		chainIndex: "56",
		httpClient: server.Client(),
	}

	to := time.Now()
	_, err := client.FetchTokenCandles(context.Background(), "0x123", to, "1h", 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_FetchTokenCandles_InvalidOhlcData(t *testing.T) {
	originalBaseURL := OkxBaseURL
	OkxBaseURL = "http://test-server"
	defer func() { OkxBaseURL = originalBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := okxResponse{
			Code: "0",
			Msg:  "success",
			Data: json.RawMessage(`[["invalid"]]`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	OkxBaseURL = server.URL

	client := &Client{
		chainIndex: "56",
		httpClient: server.Client(),
	}

	to := time.Now()
	_, err := client.FetchTokenCandles(context.Background(), "0x123", to, "1h", 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_FetchTokenCandles_MultipleCandles(t *testing.T) {
	originalBaseURL := OkxBaseURL
	OkxBaseURL = "http://test-server"
	defer func() { OkxBaseURL = originalBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := okxResponse{
			Code: "0",
			Msg:  "success",
			Data: json.RawMessage(`[
				["1700000000000","0.001","0.002","0.003","0.0005","1000","500","100"],
				["1700003600000","0.002","0.003","0.004","0.001","2000","1000","200"],
				["1700007200000","0.003","0.004","0.005","0.002","3000","1500","300"]
			]`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	OkxBaseURL = server.URL

	client := &Client{
		chainIndex: "56",
		httpClient: server.Client(),
	}

	to := time.Unix(1700008000, 0)
	result, err := client.FetchTokenCandles(context.Background(), "0x123", to, "1h", 100)
	if err != nil {
		t.Fatalf("FetchTokenCandles failed: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("expected at least 1 candle")
	}

	hasData := false
	for _, c := range result {
		if c.Open.GreaterThan(decimal.Zero) {
			hasData = true
			break
		}
	}
	if !hasData {
		t.Error("expected at least one candle with data")
	}
}
