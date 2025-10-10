package format

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// Price 格式化meme币价格，使用下标数字表示前导零个数
// decimals 参数可选，用于指定前导零后面的小数位数
func Price(price decimal.Decimal, decimals ...int) string {
	if price.IsZero() {
		return "0"
	}

	// 处理负数
	if price.IsNegative() {
		return "-" + Price(price.Neg(), decimals...)
	}

	// 如果价格大于等于1，直接返回常规格式
	if price.GreaterThanOrEqual(decimal.NewFromInt(1)) {
		return formatRegularPrice(price, decimals...)
	}

	// 转换为字符串进行处理，保留足够的精度
	priceStr := price.StringFixed(20)

	// 找到小数点位置
	dotIndex := strings.Index(priceStr, ".")
	if dotIndex == -1 {
		return priceStr
	}

	// 获取小数部分
	decimalPart := priceStr[dotIndex+1:]

	// 计算前导零的个数
	zeroCount := 0
	for i, char := range decimalPart {
		if char == '0' {
			zeroCount++
		} else {
			// 获取有效数字部分
			significantPart := decimalPart[i:]

			// 如果指定了decimals参数，截断到指定位数
			if len(decimals) > 0 && decimals[0] > 0 {
				maxLen := decimals[0]
				if len(significantPart) > maxLen {
					significantPart = significantPart[:maxLen]
				}
			} else {
				// 移除尾部的零
				significantPart = strings.TrimRight(significantPart, "0")
			}

			if significantPart == "" {
				return "0"
			}

			// 如果前导零少于3个，使用常规格式
			if zeroCount < 3 {
				return formatRegularPrice(price, decimals...)
			}

			// 构建格式化字符串
			return fmt.Sprintf("0.0%s%s", getSubscriptNumber(zeroCount), significantPart)
		}
	}

	return "0"
}

// getSubscriptNumber 将数字转换为下标格式
func getSubscriptNumber(num int) string {
	if num <= 0 {
		return ""
	}

	subscriptMap := map[rune]rune{
		'0': '₀', '1': '₁', '2': '₂', '3': '₃', '4': '₄',
		'5': '₅', '6': '₆', '7': '₇', '8': '₈', '9': '₉',
	}

	numStr := strconv.Itoa(num)
	var result strings.Builder

	for _, char := range numStr {
		if subscript, exists := subscriptMap[char]; exists {
			result.WriteRune(subscript)
		} else {
			result.WriteRune(char)
		}
	}

	return result.String()
}

// formatRegularPrice 格式化常规价格（价格>=1或前导零<2的情况）
func formatRegularPrice(price decimal.Decimal, decimals ...int) string {
	if len(decimals) == 0 {
		return price.String()
	}
	return price.Truncate(int32(decimals[0])).String()
}
