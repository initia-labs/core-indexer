package db

type TransactionEvent struct {
	Hash        string `json:"hash"`
	BlockHeight int64  `json:"block_height"`
	EventKey    string `json:"event_key"`
	EventValue  string `json:"event_value"`
	EventIndex  int    `json:"event_index"`
}
