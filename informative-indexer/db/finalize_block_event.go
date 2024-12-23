package db

import (
	"errors"
	"github.com/jackc/pgx/v5"
)

type Mode int

const (
	BeginBlock Mode = iota
	EndBlock
)

func (m Mode) String() string {
	switch m {
	case BeginBlock:
		return "BeginBlock"
	case EndBlock:
		return "EndBlock"
	default:
		return "Unknown"
	}
}

func ParseMode(s string) (Mode, error) {
	switch s {
	case "BeginBlock":
		return BeginBlock, nil
	case "EndBlock":
		return EndBlock, nil
	default:
		return -1, errors.New("invalid mode value")
	}
}

type FinalizeBlockEvent struct {
	BlockHeight int64  `json:"block_height"`
	EventKey    string `json:"event_key"`
	EventValue  string `json:"event_value"`
	EventIndex  int    `json:"event_index"`
	Mode        Mode   `json:"mode"`
}

func (f *FinalizeBlockEvent) Unmarshal(rows pgx.Rows) (map[string]interface{}, error) {
	err := rows.Scan(&f.BlockHeight, &f.EventKey, &f.EventValue, &f.EventIndex, &f.Mode)
	if err != nil {
		return nil, err
	}

	row := map[string]interface{}{
		"block_height": f.BlockHeight,
		"event_key":    f.EventKey,
		"event_value":  f.EventValue,
		"event_index":  f.EventIndex,
		"mode":         f.Mode,
	}

	return row, nil
}
