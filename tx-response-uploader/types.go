package main

const (
	ServiceName                               string = "tx-response-uploader"
	NewLcdTxResponseClaimCheckKafkaMessageKey string = "NEW_LCD_TX_RESPONSE_CLAIM_CHECK"
	HeaderHashKey                             string = "tx_hash"
	HeaderHeightKey                           string = "height"
)

type ClaimCheckMessage struct {
	ObjectPath string `json:"object_path"`
}
