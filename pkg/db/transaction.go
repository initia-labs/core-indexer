package db

import "fmt"

type Transaction struct {
	ID          string           `json:"id"`
	Hash        []byte           `json:"hash"`
	BlockHeight int64            `json:"block_height"`
	BlockIndex  int              `json:"block_index"`
	GasUsed     int64            `json:"gas_used"`
	GasLimit    uint64           `json:"gas_limit"`
	GasFee      string           `json:"gas_fee"`
	ErrMsg      *string          `json:"err_msg"`
	Success     bool             `json:"success"`
	Sender      string           `json:"sender"`
	Memo        string           `json:"memo"`
	Messages    []map[string]any `json:"messages"`
}

func GetTxID(hash string, blockHeight int64) string {
	return fmt.Sprintf("%s/%d", hash, blockHeight)
}
