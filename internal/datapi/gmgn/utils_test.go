package gmgn

import "testing"

func TestChainIdToChainName(t *testing.T) {
	tests := []struct {
		chainId    int64
		expected   string
		shouldFail bool
	}{
		{56, "bsc", false},
		{8453, "base", false},
		{4663, "robinhood", false},
		{99999, "", true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result, ok := ChainIdToChainName(tt.chainId)
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