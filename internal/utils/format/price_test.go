package format

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestFormatMemePrice(t *testing.T) {
	tests := []struct {
		name     string
		price    string // 使用字符串创建decimal，避免精度问题
		decimals []int
		expected string
	}{
		{
			name:     "零价格",
			price:    "0",
			expected: "0",
		},
		{
			name:     "负数价格",
			price:    "-0.00001234",
			expected: "-0.0₄1234",
		},
		{
			name:     "大于等于1的价格",
			price:    "1.234567",
			expected: "1.234567",
		},
		{
			name:     "大于等于1的价格并且指定小数位数",
			price:    "1.234567",
			decimals: []int{3},
			expected: "1.234",
		},
		{
			name:     "大于1的价格",
			price:    "123.456789",
			expected: "123.456789",
		},
		{
			name:     "前导零少于2个的小数",
			price:    "0.1234",
			expected: "0.1234",
		},
		{
			name:     "前导零等于3个的小数",
			price:    "0.0001234",
			expected: "0.0₃1234",
		},
		{
			name:     "前导零多个的小数",
			price:    "0.00001234",
			expected: "0.0₄1234",
		},
		{
			name:     "前导零很多的小数",
			price:    "0.000000001234",
			expected: "0.0₈1234",
		},
		{
			name:     "指定小数位数",
			price:    "0.00001234567",
			decimals: []int{3},
			expected: "0.0₄123",
		},
		{
			name:     "尾部有零的情况",
			price:    "0.000012340000",
			expected: "0.0₄1234",
		},
		{
			name:     "只有前导零的情况",
			price:    "0.000000000",
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, err := decimal.NewFromString(tt.price)
			if err != nil {
				t.Fatalf("创建decimal失败: %v", err)
			}

			var result string
			if len(tt.decimals) > 0 {
				result = Price(price, tt.decimals[0])
			} else {
				result = Price(price)
			}

			if result != tt.expected {
				t.Errorf("Price() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFormatRegularPrice(t *testing.T) {
	tests := []struct {
		name     string
		price    string
		expected string
	}{
		{
			name:     "大于1的价格",
			price:    "123.456789123",
			expected: "123.456789123",
		},
		{
			name:     "等于1的价格",
			price:    "1.0",
			expected: "1",
		},
		{
			name:     "小于1的价格",
			price:    "0.12345678901",
			expected: "0.12345678901",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, err := decimal.NewFromString(tt.price)
			if err != nil {
				t.Fatalf("创建decimal失败: %v", err)
			}

			result := formatRegularPrice(price)
			if result != tt.expected {
				t.Errorf("formatRegularPrice() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetSubscriptNumber(t *testing.T) {
	tests := []struct {
		name     string
		num      int
		expected string
	}{
		{
			name:     "零或负数",
			num:      0,
			expected: "",
		},
		{
			name:     "负数",
			num:      -1,
			expected: "",
		},
		{
			name:     "单个数字",
			num:      1,
			expected: "₁",
		},
		{
			name:     "多个数字",
			num:      123,
			expected: "₁₂₃",
		},
		{
			name:     "包含所有数字",
			num:      1234567890,
			expected: "₁₂₃₄₅₆₇₈₉₀",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSubscriptNumber(tt.num)
			if result != tt.expected {
				t.Errorf("getSubscriptNumber() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// 边界情况测试
func TestEdgeCases(t *testing.T) {
	t.Run("极小的价格", func(t *testing.T) {
		price, _ := decimal.NewFromString("0.000000000000000001")
		result := Price(price)
		// 应该能正常处理而不崩溃
		if result == "" {
			t.Error("极小价格处理失败")
		}
	})

	t.Run("极大的价格", func(t *testing.T) {
		price, _ := decimal.NewFromString("999999999999.123456")
		result := Price(price)
		expected := "999999999999.123456"
		if result != expected {
			t.Errorf("极大价格处理失败, got %v, expected %v", result, expected)
		}
	})

	t.Run("精确的1.0", func(t *testing.T) {
		price := decimal.NewFromInt(1)
		result := Price(price)
		expected := "1"
		if result != expected {
			t.Errorf("精确1.0处理失败, got %v, expected %v", result, expected)
		}
	})

	t.Run("无限小数", func(t *testing.T) {
		// 1/3 的近似值
		price := decimal.NewFromFloat(1.0 / 3.0)
		result := Price(price)
		// 应该能正常处理而不崩溃
		if result == "" {
			t.Error("无限小数处理失败")
		}
	})
}
