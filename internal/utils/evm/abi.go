package evm

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const erc20ABI = `[
	{
		"constant": true,
		"inputs": [],
		"name": "name",
		"outputs": [{"name": "", "type": "string"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "symbol",
		"outputs": [{"name": "", "type": "string"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "decimals",
		"outputs": [{"name": "", "type": "uint8"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [{"name": "_owner", "type": "address"}],
		"name": "balanceOf",
		"outputs": [{"name": "balance", "type": "uint256"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{"name": "_owner", "type": "address"},
			{"name": "_spender", "type": "address"}
		],
		"name": "allowance",
		"outputs": [{"name": "", "type": "uint256"}],
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_spender", "type": "address"},
			{"name": "_value", "type": "uint256"}
		],
		"name": "approve",
		"outputs": [{"name": "", "type": "bool"}],
		"type": "function"
	}
]`

var (
	ERC20ABI abi.ABI
)

func init() {
	var err error
	ERC20ABI, err = abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		panic(fmt.Errorf("failed to parse ABI: %w", err))
	}
}
