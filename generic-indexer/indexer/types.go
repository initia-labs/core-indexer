package indexer

type LCDTxResponses struct {
	Height int64          `json:"height"`
	Hash   string         `json:"hash"`
	Result map[string]any `json:"result"`
}
