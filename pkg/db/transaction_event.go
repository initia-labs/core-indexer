package db

import "github.com/jackc/pgx/v5"

type TransactionEvent struct {
	TransactionHash string `json:"transaction_hash"`
	BlockHeight     int64  `json:"block_height"`
	EventKey        string `json:"event_key"`
	EventValue      string `json:"event_value"`
	EventIndex      int    `json:"event_index"`
}

func (t *TransactionEvent) Unmarshal(rows pgx.Rows) (map[string]any, error) {
	err := rows.Scan(&t.TransactionHash, &t.BlockHeight, &t.EventKey, &t.EventValue, &t.EventIndex)
	if err != nil {
		return nil, err
	}

	row := map[string]any{
		"transaction_hash": t.TransactionHash,
		"block_height":     t.BlockHeight,
		"event_key":        t.EventKey,
		"event_value":      t.EventValue,
		"event_index":      t.EventIndex,
	}

	return row, nil
}
