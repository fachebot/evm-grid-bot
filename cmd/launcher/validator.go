package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/fachebot/evm-grid-bot/internal/dexagg/okxweb3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Validator 配置验证器
type Validator struct{}

// RPCValidationResult RPC验证结果
type RPCValidationResult struct {
	ChainID int64
	Error   error
}

// ValidateRPC 验证RPC URL
func (v *Validator) ValidateRPC(rpcUrl string, expectedChainId int64) error {
	result := v.ValidateRPCWithResult(rpcUrl, expectedChainId)
	return result.Error
}

// ValidateRPCWithResult 验证RPC URL并返回链ID
func (v *Validator) ValidateRPCWithResult(rpcUrl string, expectedChainId int64) RPCValidationResult {
	ctx := context.Background()
	
	// 创建RPC客户端
	rpcClient, err := rpc.DialContext(ctx, rpcUrl)
	if err != nil {
		return RPCValidationResult{
			Error: fmt.Errorf("无法连接到RPC服务器: %w", err),
		}
	}
	defer rpcClient.Close()

	// 创建以太坊客户端
	ethClient := ethclient.NewClient(rpcClient)
	defer ethClient.Close()

	// 验证链ID
	chainId, err := ethClient.ChainID(ctx)
	if err != nil {
		return RPCValidationResult{
			Error: fmt.Errorf("无法获取链ID: %w", err),
		}
	}

	chainIdInt64 := chainId.Int64()
	if chainIdInt64 != expectedChainId {
		return RPCValidationResult{
			ChainID: chainIdInt64,
			Error:   fmt.Errorf("链ID不匹配: 期望 %d, 实际 %d", expectedChainId, chainIdInt64),
		}
	}

	return RPCValidationResult{
		ChainID: chainIdInt64,
		Error:   nil,
	}
}

// ValidateOKX 验证OKX API密钥
func (v *Validator) ValidateOKX(apikey, secretkey, passphrase string) error {
	ctx := context.Background()
	
	// 创建OKX客户端（使用dexagg包中的客户端）
	client := okxweb3.NewClient(apikey, secretkey, passphrase, nil)
	
	// 调用简单API验证
	_, err := client.GetSupportedChains(ctx)
	if err != nil {
		return fmt.Errorf("OKX API验证失败: %w", err)
	}

	return nil
}

// TelegramValidationResult Telegram验证结果
type TelegramValidationResult struct {
	Username string
	Error    error
}

// ValidateTelegram 验证Telegram Bot Token
func (v *Validator) ValidateTelegram(token string) error {
	result := v.ValidateTelegramWithResult(token)
	return result.Error
}

// ValidateTelegramWithResult 验证Telegram Bot Token并返回bot信息
func (v *Validator) ValidateTelegramWithResult(token string) TelegramValidationResult {
	// 创建Telegram Bot API客户端
	botApi, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return TelegramValidationResult{
			Error: fmt.Errorf("无法创建Telegram Bot客户端: %w", err),
		}
	}

	// 验证token并获取bot信息
	bot, err := botApi.GetMe()
	if err != nil {
		return TelegramValidationResult{
			Error: fmt.Errorf("Telegram Bot Token验证失败: %w", err),
		}
	}

	return TelegramValidationResult{
		Username: bot.UserName,
		Error:    nil,
	}
}

// OpenURL 在默认浏览器中打开URL
func OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
