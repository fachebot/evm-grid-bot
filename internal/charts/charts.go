package charts

import (
	"fmt"
	"strconv"
	"time"

	"github.com/markcheno/go-talib"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

type Ohlc struct {
	Open   decimal.Decimal `json:"open"`
	Close  decimal.Decimal `json:"close"`
	High   decimal.Decimal `json:"high"`
	Low    decimal.Decimal `json:"low"`
	Time   time.Time       `json:"time"`
	Volume decimal.Decimal `json:"volume"`
}

type TokenOhlcs struct {
	Token string
	Ohlcs []Ohlc
}

func CalculateRSI(ohlcs []Ohlc) []float64 {
	prices := lo.Map(ohlcs, func(ohlc Ohlc, idx int) float64 {
		return ohlc.Close.InexactFloat64()
	})
	return talib.Rsi(prices, 14)
}

func FillMissingOhlc(tokenOhlcs []Ohlc, to time.Time, interval time.Duration) []Ohlc {
	var filled []Ohlc
	if len(tokenOhlcs) == 0 {
		return tokenOhlcs
	}

	prev := tokenOhlcs[0]
	filled = append(filled, prev)

	for i := 1; i < len(tokenOhlcs); i++ {
		curr := tokenOhlcs[i]
		missingCount := int(curr.Time.Sub(prev.Time) / interval)

		for j := 1; j < missingCount; j++ {
			missingTime := prev.Time.Add(time.Duration(j) * interval)
			missingOhlc := Ohlc{
				Open:   prev.Close,
				Close:  prev.Close,
				High:   prev.Close,
				Low:    prev.Close,
				Time:   missingTime,
				Volume: decimal.Zero,
			}
			filled = append(filled, missingOhlc)
		}
		filled = append(filled, curr)
		prev = curr
	}

	missingCount := int(to.Sub(prev.Time) / interval)
	for i := 0; i < missingCount; i++ {
		missingTime := prev.Time.Add(time.Duration(i) * interval)
		missingOhlc := Ohlc{
			Open:   prev.Close,
			Close:  prev.Close,
			High:   prev.Close,
			Low:    prev.Close,
			Time:   missingTime,
			Volume: decimal.Zero,
		}
		filled = append(filled, missingOhlc)
	}

	return filled
}

func ResolutionToDuration(resolution string) (time.Duration, error) {
	if len(resolution) < 2 {
		return 0, fmt.Errorf("invalid resolution format: %q (expected format: <number><unit> where unit is s, m, h, d)", resolution)
	}

	unit := resolution[len(resolution)-1]
	nStr := resolution[:len(resolution)-1]

	n, err := strconv.Atoi(nStr)
	if err != nil {
		return 0, fmt.Errorf("invalid number part %q: %v", nStr, err)
	}

	switch unit {
	case 's':
		return time.Duration(n) * time.Second, nil
	case 'm':
		return time.Duration(n) * time.Minute, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * time.Hour * 24, nil
	default:
		return 0, fmt.Errorf("invalid unit %q (expected s, m, h, d)", string(unit))
	}
}
