package db

type TransactionEvent struct {
	TransactionHash string `json:"transaction_hash"`
	BlockHeight     int64  `json:"block_height"`
	EventKey        string `json:"event_key"`
	EventValue      string `json:"event_value"`
	EventIndex      int    `json:"event_index"`
}
