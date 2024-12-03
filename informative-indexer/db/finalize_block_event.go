package db

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

type FinalizeBlockEvent struct {
	BlockHeight int64  `json:"block_height"`
	EventKey    string `json:"event_key"`
	EventValue  string `json:"event_value"`
	EventIndex  int    `json:"event_index"`
	Mode        Mode   `json:"mode"`
}
