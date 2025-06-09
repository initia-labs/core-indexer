package dto

import "encoding/json"

type BlockHeightLatestResponse struct {
	Height int64 `json:"height"`
}

type BlockTimeAverageResponse struct {
	AverageBlockTime float64 `json:"avg_block_time"`
}

type BlockModel struct {
	Hash            string `json:"hash"`
	Height          int64  `json:"height"`
	Timestamp       string `json:"timestamp"`
	OperatorAddress string `json:"operator_address"`
	Moniker         string `json:"moniker"`
	Identity        string `json:"identity"`
	TxCount         int64  `json:"tx_count"`
}

type BlockProposerResponse struct {
	Identify        string `json:"identify"`
	Moniker         string `json:"moniker"`
	OperatorAddress string `json:"operator_address"`
}

type BlockResponse struct {
	Hash      string                `json:"hash"`
	Height    int64                 `json:"height"`
	Timestamp string                `json:"timestamp"`
	TxCount   int64                 `json:"tx_count"`
	Proposer  BlockProposerResponse `json:"proposer"`
}

type BlocksResponse struct {
	Items      []BlockResponse    `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

type BlockInfoModel struct {
	Hash            string `json:"hash"`
	Height          int64  `json:"height"`
	Timestamp       string `json:"timestamp"`
	OperatorAddress string `json:"operator_address"`
	Moniker         string `json:"moniker"`
	Identity        string `json:"identity"`
	GasUsed         int64  `json:"gas_used"`
	GasLimit        int64  `json:"gas_limit"`
}

type BlockInfoResponse struct {
	GasLimit  int64                 `json:"gas_limit"`
	GasUsed   int64                 `json:"gas_used"`
	Hash      string                `json:"hash"`
	Height    int64                 `json:"height"`
	Timestamp string                `json:"timestamp"`
	Proposer  BlockProposerResponse `json:"proposer"`
}

type BlockTxResponse struct {
	Height    int64           `json:"height"`
	Timestamp string          `json:"timestamp"`
	Address   string          `json:"address"`
	Hash      string          `json:"hash"`
	Success   bool            `json:"success"`
	Messages  json.RawMessage `json:"messages" swaggertype:"object"`
	IsSend    bool            `json:"is_send"`
	IsIbc     bool            `json:"is_ibc"`
	IsOpinit  bool            `json:"is_opinit"`
}

type BlockTxsResponse struct {
	Items      []BlockTxResponse  `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}
