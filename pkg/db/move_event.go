package db

import (
	"github.com/jackc/pgx/v5"
)

type MoveEvent struct {
	TypeTag         string `json:"type_tag"`
	Data            string `json:"data"`
	BlockHeight     int64  `json:"block_height"`
	TransactionHash string `json:"transaction_hash"`
	EventIndex      int    `json:"event_index"`
}

func (m *MoveEvent) Unmarshal(rows pgx.Rows) (map[string]interface{}, error) {
	err := rows.Scan(&m.TypeTag, &m.Data, &m.BlockHeight, &m.TransactionHash, &m.EventIndex)
	if err != nil {
		return nil, err
	}

	row := map[string]interface{}{
		"type_tag":         m.TypeTag,
		"data":             m.Data,
		"block_height":     m.BlockHeight,
		"transaction_hash": m.TransactionHash,
		"event_index":      m.EventIndex,
	}

	return row, nil
}
