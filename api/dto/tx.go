package dto

// Coins represents a list of coin amounts
type Coins []Coin

// Coin represents a single coin amount
type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

// Log represents a transaction log
type Log struct {
	MsgIndex int64       `json:"msg_index"`
	Log      interface{} `json:"log"` // Can be string or map[string]string
	Events   []Event     `json:"events,omitempty"`
}

// Event represents a transaction event
type Event struct {
	Type       string           `json:"type"`
	Attributes []EventAttribute `json:"attributes"`
}

// EventAttribute represents a key-value pair in an event
type EventAttribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Body represents the transaction body
type Body struct {
	Messages      []map[string]interface{} `json:"messages"`
	Memo          string                   `json:"memo"`
	TimeoutHeight string                   `json:"timeout_height,omitempty"`
}

// Fee represents the transaction fee
type Fee struct {
	Amount   Coins  `json:"amount"`
	GasLimit string `json:"gas_limit"`
	Payer    string `json:"payer"`
	Granter  string `json:"granter"`
}

// PublicKey represents a public key
type PublicKey struct {
	Type string `json:"@type"`
	Key  string `json:"key"`
}

// SignerInfo represents information about a transaction signer
type SignerInfo struct {
	PublicKey PublicKey `json:"public_key"`
	Sequence  string    `json:"sequence"`
}

// AuthInfo represents the transaction authentication information
type AuthInfo struct {
	SignerInfos []SignerInfo `json:"signer_infos"`
	Fee         Fee          `json:"fee"`
}

// RestTx represents the raw transaction data
type RestTx struct {
	Body       Body     `json:"body"`
	AuthInfo   AuthInfo `json:"auth_info"`
	Signatures []string `json:"signatures"`
}

// RestTxResponse represents the complete transaction response
type RestTxResponse struct {
	Tx         RestTx     `json:"tx"`
	TxResponse TxResponse `json:"tx_response"`
}

// TxResponse represents the response structure for a transaction
type TxResponse struct {
	Height    string  `json:"height"`
	TxHash    string  `json:"txhash"`
	Codespace string  `json:"codespace,omitempty"`
	Code      int     `json:"code,omitempty"`
	RawLog    string  `json:"raw_log"`
	Logs      []Log   `json:"logs"`
	GasWanted string  `json:"gas_wanted"`
	GasUsed   string  `json:"gas_used"`
	Tx        RestTx  `json:"tx"`
	Timestamp string  `json:"timestamp"` // unix time (GMT)
	Events    []Event `json:"events"`
}
