package flusher

type LCDTxResponses struct {
	Height int64          `json:"height"`
	Hash   string         `json:"hash"`
	Result map[string]any `json:"result"`
}

type RPCEndpoints struct {
	RPCs []RPC `json:"rpcs"`
}

type RPC struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}
