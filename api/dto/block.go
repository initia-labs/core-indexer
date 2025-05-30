package dto

type RestBlockHeightLatestResponse struct {
	Height int64 `json:"height"`
}

type RestBlockTimeAverageResponse struct {
	AverageBlockTime float64 `json:"avg_block_time"`
}
