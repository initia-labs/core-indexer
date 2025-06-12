package utils

import (
	"sort"
)

func Median(nums []float64) float64 {
	n := len(nums)
	if n == 0 {
		return 0
	}

	sorted := make([]float64, n)
	copy(sorted, nums)
	sort.Float64s(sorted)

	mid := n / 2
	if n%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}
