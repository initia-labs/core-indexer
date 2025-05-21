package txparser

import (
	"encoding/json"
	"fmt"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/initia-labs/initia/app/params"
)

type intoAny interface {
	AsAny() *codectypes.Any
}

func ParseTxMessages(messages []types.Msg, md []map[string]any) []map[string]any {
	var parsedMessages []map[string]any
	for idx, msg := range messages {
		parsedMessages = append(parsedMessages, map[string]any{
			"type":   types.MsgTypeURL(msg),
			"detail": md[idx],
		})
	}

	return parsedMessages
}

// mkTxResult returns a sdk.TxResponse from the given Tendermint ResultTx.
func mkTxResult(txConfig client.TxConfig, resTx *coretypes.ResultTx, blockTime time.Time) (*types.TxResponse, error) {
	txb, err := txConfig.TxDecoder()(resTx.Tx)
	if err != nil {
		return nil, err
	}
	p, ok := txb.(intoAny)
	if !ok {
		return nil, fmt.Errorf("expecting a type implementing intoAny, got: %T", txb)
	}
	asAny := p.AsAny()
	return types.NewResponseResultTx(resTx, asAny, blockTime.Format(time.RFC3339)), nil
}

// ParseMessageDicts returns an array of JsDict decoded version for messages in the provided transaction.
func ParseMessageDicts(txResJsDict map[string]any) []map[string]any {
	details := make([]map[string]any, 0)
	tx := txResJsDict["tx"].(map[string]any)
	body := tx["body"].(map[string]any)
	msgs := body["messages"].([]any)
	for _, msg := range msgs {
		detail := msg.(map[string]any)
		details = append(details, detail)
	}
	return details
}

// GetTxResponse converts a transaction result into a JSON-serializable map and proto transaction.
// It returns an error instead of panicking to allow proper error handling by the caller.
func GetTxResponse(encodingConfig params.EncodingConfig, blockTime time.Time, resTx coretypes.ResultTx) (map[string]any, txtypes.Tx, error) {
	// Input validation
	if resTx.Tx == nil {
		return nil, txtypes.Tx{}, fmt.Errorf("transaction cannot be nil")
	}

	// Create transaction result
	txResult, err := mkTxResult(encodingConfig.TxConfig, &resTx, blockTime)
	if err != nil {
		return nil, txtypes.Tx{}, fmt.Errorf("failed to create tx result: %w", err)
	}

	// Get proto transaction
	cachedValue := txResult.Tx.GetCachedValue()
	if cachedValue == nil {
		return nil, txtypes.Tx{}, fmt.Errorf("cached transaction value is nil")
	}

	protoTx, ok := cachedValue.(*txtypes.Tx)
	if !ok {
		return nil, txtypes.Tx{}, fmt.Errorf("failed to convert to proto transaction: unexpected type %T", cachedValue)
	}

	// Create and marshal response
	response := &txtypes.GetTxResponse{
		Tx:         protoTx,
		TxResponse: txResult,
	}

	txResJson, err := codec.ProtoMarshalJSON(response, nil)
	if err != nil {
		return nil, txtypes.Tx{}, fmt.Errorf("failed to marshal transaction response: %w", err)
	}

	// Unmarshal to map
	var txResJsDict map[string]any
	if err := json.Unmarshal(txResJson, &txResJsDict); err != nil {
		return nil, txtypes.Tx{}, fmt.Errorf("failed to unmarshal transaction response: %w", err)
	}

	return txResJsDict, *protoTx, nil
}
