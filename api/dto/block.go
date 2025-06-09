package dto

type BlockHeightLatestResponse struct {
	Height int64 `json:"height"`
}

type BlockTimeAverageResponse struct {
	AverageBlockTime float64 `json:"avg_block_time"`
}
