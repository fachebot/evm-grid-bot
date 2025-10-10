package utils

import "fmt"

func GetNetworkName(chainId int64) string {
	switch chainId {
	case 56:
		return "BSC"
	case 8453:
		return "Base"
	}
	return ""
}

func GetOkxTokenLink(chainId int64, token string) string {
	switch chainId {
	case 56:
		return fmt.Sprintf("https://web3.okx.com/zh-hant/token/bsc/%s", token)
	case 8453:
		return fmt.Sprintf("https://web3.okx.com/zh-hant/token/base/%s", token)
	}
	return ""
}

func GetOkxAccountLink(chainId int64, account string) string {
	switch chainId {
	case 56:
		return fmt.Sprintf("https://web3.okx.com/zh-hant/portfolio/%s/analysis?chainIndex=56", account)
	case 8453:
		return fmt.Sprintf("https://web3.okx.com/zh-hant/portfolio/%s/analysis?chainIndex=8453", account)
	}
	return ""
}

func GetGmgnTokenLink(chainId int64, token string) string {
	switch chainId {
	case 56:
		return fmt.Sprintf("https://gmgn.ai/bsc/token/%s", token)
	case 8453:
		return fmt.Sprintf("https://gmgn.ai/base/token/%s", token)
	}
	return ""
}

func GetGmgnAccountLink(chainId int64, account string) string {
	switch chainId {
	case 56:
		return fmt.Sprintf("https://gmgn.ai/bsc/address/%s", account)
	case 8453:
		return fmt.Sprintf("https://gmgn.ai/base/address/%s", account)
	}
	return ""
}

func GetDexscreenerTokenLink(chainId int64, token string) string {
	switch chainId {
	case 56:
		return fmt.Sprintf("https://dexscreener.com/bsc/%s", token)
	case 8453:
		return fmt.Sprintf("https://dexscreener.com/base/%s", token)
	}
	return ""
}

func GetBlockExplorerTxLink(chainId int64, hash string) string {
	switch chainId {
	case 56:
		return fmt.Sprintf("https://bscscan.com/tx/%s", hash)
	case 8453:
		return fmt.Sprintf("https://basescan.org/tx/%s", hash)
	}
	return ""
}

func GetBlockExplorerTokenLink(chainId int64, token string) string {
	switch chainId {
	case 56:
		return fmt.Sprintf("https://bscscan.com/token/%s", token)
	case 8453:
		return fmt.Sprintf("https://basescan.org/token/%s", token)
	}
	return ""
}

func GetBlockExplorerAccountLink(chainId int64, account string) string {
	switch chainId {
	case 56:
		return fmt.Sprintf("https://bscscan.com/address/%s", account)
	case 8453:
		return fmt.Sprintf("https://basescan.org/address/%s", account)
	}
	return ""
}
