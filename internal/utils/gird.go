package utils

import (
	"errors"
	"slices"

	"github.com/shopspring/decimal"
)

func GenerateGrid(lowerPriceBound, upperPriceBound, takeProfitRatio decimal.Decimal) ([]decimal.Decimal, error) {
	if lowerPriceBound.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("lower price bound must be positive")
	}
	if upperPriceBound.LessThanOrEqual(lowerPriceBound) {
		return nil, errors.New("upper price bound must be greater than lower price bound")
	}
	if takeProfitRatio.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("take profit ratio must be positive")
	}

	grid := lowerPriceBound
	result := make([]decimal.Decimal, 0)

	for grid.LessThan(upperPriceBound) {
		result = append(result, grid)
		grid = grid.Add(grid.Mul(takeProfitRatio))
	}
	return result, nil
}

func CalculateGridPosition(gridList []decimal.Decimal, price decimal.Decimal) (int, bool) {
	if len(gridList) == 0 {
		return 0, false
	}

	sortedGrid := make([]decimal.Decimal, len(gridList))
	copy(sortedGrid, gridList)
	slices.SortFunc(sortedGrid, func(a, b decimal.Decimal) int {
		return a.Cmp(b)
	})

	if price.LessThan(sortedGrid[0]) {
		return 0, true
	}

	if price.GreaterThan(sortedGrid[len(sortedGrid)-1]) {
		return 0, false
	}

	for idx := range len(sortedGrid) - 1 {
		if price.GreaterThanOrEqual(sortedGrid[idx]) && price.LessThanOrEqual(sortedGrid[idx+1]) {
			return idx + 1, true
		}
	}

	return 0, false
}
