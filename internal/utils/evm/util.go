package evm

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

var (
	// MaxInt256 represents the maximum value for int256 (2^255 - 1)
	MaxInt256 *big.Int

	// MaxUint256 represents the maximum value for uint256 (2^256 - 1)
	MaxUint256 *big.Int
)

func init() {
	MaxInt256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1))
	MaxUint256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
}

func ParseETH(value *big.Int) decimal.Decimal {
	return ParseUnits(value, 18)
}

func ParseUnits(value *big.Int, decimals uint8) decimal.Decimal {
	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromInt32(int32(decimals)))
	num, _ := decimal.NewFromString(value.String())
	result := num.DivRound(mul, int32(decimals)).Truncate(int32(decimals))
	return result
}

func FormatETH(amount decimal.Decimal) *big.Int {
	return FormatUnits(amount, 18)
}

func FormatUnits(amount decimal.Decimal, decimals uint8) *big.Int {
	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromInt32(int32(decimals)))
	result := amount.Mul(mul)

	wei := big.NewInt(0)
	wei.SetString(result.String(), 10)
	return wei
}

func EncodeERC20ApproveInput(spender string, amount *big.Int) ([]byte, error) {
	if spender == "" {
		return nil, errors.New("spender address cannot be empty")
	}
	if amount == nil {
		return nil, errors.New("amount cannot be nil")
	}

	spenderAddr := common.HexToAddress(spender)
	data, err := ERC20ABI.Pack("approve", spenderAddr, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to pack approve call: %w", err)
	}

	return data, nil
}

func DecodeERC20ApproveInput(input []byte) (spender string, amount *big.Int, err error) {
	if len(input) < 4 {
		return "", nil, errors.New("input data too short")
	}

	approveMethodID := ERC20ABI.Methods["approve"].ID
	if !bytes.Equal(input[:4], approveMethodID) {
		return "", nil, errors.New("input data is not for approve function")
	}

	values, err := ERC20ABI.Methods["approve"].Inputs.Unpack(input[4:])
	if err != nil {
		return "", nil, fmt.Errorf("failed to unpack approve input: %w", err)
	}

	if len(values) != 2 {
		return "", nil, fmt.Errorf("expected 2 parameters, got %d", len(values))
	}

	spenderAddr, ok := values[0].(common.Address)
	if !ok {
		return "", nil, errors.New("failed to parse spender address")
	}

	amountValue, ok := values[1].(*big.Int)
	if !ok {
		return "", nil, errors.New("failed to parse amount")
	}

	return spenderAddr.Hex(), amountValue, nil
}

func GetAddress(prv *ecdsa.PrivateKey) (common.Address, error) {
	publicKey := prv.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return address, nil
}

func GetBalance(ctx context.Context, ethClient *ethclient.Client, ownerAddress string) (*big.Int, error) {
	return ethClient.BalanceAt(ctx, common.HexToAddress(ownerAddress), nil)
}

func GetTokenMeta(ctx context.Context, ethClient *ethclient.Client, tokenAddress string) (*Metadata, error) {
	tokenAddr := common.HexToAddress(tokenAddress)

	// 获取代币名称
	nameData, err := ERC20ABI.Pack("name")
	if err != nil {
		return nil, fmt.Errorf("failed to pack name call: %w", err)
	}

	nameResult, err := ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &tokenAddr,
		Data: nameData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call name: %w", err)
	}

	var name string
	err = ERC20ABI.UnpackIntoInterface(&name, "name", nameResult)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack name: %w", err)
	}

	// 获取代币符号
	symbolData, err := ERC20ABI.Pack("symbol")
	if err != nil {
		return nil, fmt.Errorf("failed to pack symbol call: %w", err)
	}

	symbolResult, err := ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &tokenAddr,
		Data: symbolData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call symbol: %w", err)
	}

	var symbol string
	err = ERC20ABI.UnpackIntoInterface(&symbol, "symbol", symbolResult)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack symbol: %w", err)
	}

	// 获取代币精度
	decimalsData, err := ERC20ABI.Pack("decimals")
	if err != nil {
		return nil, fmt.Errorf("failed to pack decimals call: %w", err)
	}

	decimalsResult, err := ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &tokenAddr,
		Data: decimalsData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call decimals: %w", err)
	}

	var decimals uint8
	err = ERC20ABI.UnpackIntoInterface(&decimals, "decimals", decimalsResult)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack decimals: %w", err)
	}

	return &Metadata{
		Name:     name,
		Symbol:   symbol,
		Decimals: decimals,
	}, nil
}

func GetTokenBalance(ctx context.Context, ethClient *ethclient.Client, tokenAddress, ownerAddress string) (*big.Int, error) {
	tokenAddr := common.HexToAddress(tokenAddress)
	ownerAddr := common.HexToAddress(ownerAddress)

	data, err := ERC20ABI.Pack("balanceOf", ownerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to pack balanceOf call: %w", err)
	}

	result, err := ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &tokenAddr,
		Data: data,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call balanceOf: %w", err)
	}

	var balance *big.Int
	err = ERC20ABI.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack balanceOf result: %w", err)
	}

	return balance, nil
}

func GetTokenAllowance(ctx context.Context, ethClient *ethclient.Client, tokenAddress, ownerAddress, spenderAddress string) (*big.Int, error) {
	tokenAddr := common.HexToAddress(tokenAddress)
	ownerAddr := common.HexToAddress(ownerAddress)
	spenderAddr := common.HexToAddress(spenderAddress)

	data, err := ERC20ABI.Pack("allowance", ownerAddr, spenderAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to pack allowance call: %w", err)
	}

	result, err := ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &tokenAddr,
		Data: data,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call allowance: %w", err)
	}

	var allowance *big.Int
	err = ERC20ABI.UnpackIntoInterface(&allowance, "allowance", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack allowance result: %w", err)
	}

	return allowance, nil
}

func GetTokenBalanceChanges(ctx context.Context, ethClient *ethclient.Client, receipt *types.Receipt, ownerAddress string) (map[common.Address]*big.Int, error) {
	// Transfer 事件的签名
	transferEventSig := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	// 解析日志中的 Transfer 事件
	changes := make(map[common.Address]*big.Int)
	ownerAddr := common.HexToAddress(ownerAddress)
	for _, log := range receipt.Logs {
		if len(log.Topics) > 0 && log.Topics[0] == transferEventSig {
			if len(log.Topics) >= 3 {
				from := common.BytesToAddress(log.Topics[1].Bytes())
				to := common.BytesToAddress(log.Topics[2].Bytes())

				// 只关心涉及目标地址的转账
				if from == ownerAddr || to == ownerAddr {
					change := big.NewInt(0)
					amount := new(big.Int).SetBytes(log.Data)

					if from == ownerAddr {
						change.Neg(amount)
					} else {
						change.Set(amount)
					}

					v, ok := changes[log.Address]
					if !ok {
						v = change
					} else {
						v = big.NewInt(0).Add(v, change)
					}
					changes[log.Address] = v
				}
			}
		}
	}

	return changes, nil
}
