package flusher

// EvMap is a type alias for SDK events mapping from Attr.Key to the list of values.
type EvMap map[string][]string

// JsDict is a type alias for JSON dictionary.
type JsDict map[string]interface{}

type LCDTxResponses struct {
	Height int64  `json:"height"`
	Hash   string `json:"hash"`
	Result JsDict `json:"result"`
}

type RPCEndpoints struct {
	RPCs []RPC `json:"rpcs"`
}

type RPC struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}
