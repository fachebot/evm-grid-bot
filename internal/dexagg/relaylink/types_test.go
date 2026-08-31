package relaylink

import (
	"encoding/json"
	"testing"
)

// TestChainsUnmarshalFractionalFees 回归测试: relay链列表中的手续费字段可能是小数(如0.8/2.25),
// 若按int解析会导致整个chains列表反序列化失败, 进而被误报为 "unsupported chain"
func TestChainsUnmarshalFractionalFees(t *testing.T) {
	payload := `{
		"chains": [
			{
				"id": 1,
				"name": "ethereum",
				"displayName": "Ethereum",
				"httpRpcUrl": "https://rpc.ankr.com/eth",
				"wsRpcUrl": "",
				"explorerUrl": "https://etherscan.io",
				"explorerName": "Etherscan",
				"depositEnabled": true,
				"tokenSupport": "all",
				"disabled": false,
				"partialDisableLimit": 0,
				"blockProductionLagging": false,
				"currency": {
					"id": "eth",
					"symbol": "ETH",
					"name": "Ether",
					"address": "0x0000000000000000000000000000000000000000",
					"decimals": 18,
					"supportsBridging": true
				},
				"withdrawalFee": 0.8,
				"depositFee": 0,
				"surgeEnabled": false,
				"vmType": "evm",
				"baseChainId": 1,
				"erc20Currencies": [
					{
						"id": "usdc",
						"symbol": "USDC",
						"name": "USD Coin",
						"address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						"decimals": 6,
						"supportsBridging": true,
						"supportsPermit": true,
						"withdrawalFee": 0.8,
						"depositFee": 0,
						"surgeEnabled": false
					}
				]
			}
		]
	}`

	var chains Chains
	if err := json.Unmarshal([]byte(payload), &chains); err != nil {
		t.Fatalf("unmarshal chains failed: %v", err)
	}
	if len(chains.Chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains.Chains))
	}

	chain := chains.Chains[0]
	if chain.WithdrawalFee != 0.8 {
		t.Errorf("expected chain withdrawalFee 0.8, got %v", chain.WithdrawalFee)
	}
	if len(chain.ERC20Currencies) != 1 {
		t.Fatalf("expected 1 erc20 currency, got %d", len(chain.ERC20Currencies))
	}
	if chain.ERC20Currencies[0].WithdrawalFee != 0.8 {
		t.Errorf("expected erc20 withdrawalFee 0.8, got %v", chain.ERC20Currencies[0].WithdrawalFee)
	}
}