package txparser

import (
	"encoding/json"
	"fmt"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/initia-labs/initia/app/params"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

// intoAny is an interface that defines a method to convert a type into a protobuf Any message.
// This is used for serializing transaction data into a format that can be included in responses.
type intoAny interface {
	AsAny() *codectypes.Any
}

func ParseTxMessages(messages []sdk.Msg, md []map[string]any) []map[string]any {
	var parsedMessages []map[string]any
	for idx, msg := range messages {
		parsedMessages = append(parsedMessages, map[string]any{
			"type":   sdk.MsgTypeURL(msg),
			"detail": md[idx],
		})
	}

	return parsedMessages
}

// mkTxResult returns a sdk.TxResponse from the given Tendermint ResultTx.
func mkTxResult(txConfig client.TxConfig, resTx *coretypes.ResultTx, blockTime time.Time) (*sdk.TxResponse, error) {
	txb, err := txConfig.TxDecoder()(resTx.Tx)
	if err != nil {
		return nil, err
	}
	p, ok := txb.(intoAny)
	if !ok {
		return nil, fmt.Errorf("expecting a type implementing intoAny, got: %T", txb)
	}
	asAny := p.AsAny()
	return sdk.NewResponseResultTx(resTx, asAny, blockTime.Format(time.RFC3339)), nil
}

// parseMessages returns an array of JsDict decoded version for messages in the provided transaction.
func parseMessages(txResJsDict map[string]any) ([]map[string]any, error) {
	// Validate input
	if txResJsDict == nil {
		return nil, fmt.Errorf("transaction response cannot be nil")
	}

	// Safely extract tx field
	tx, ok := txResJsDict["tx"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid transaction format: missing or invalid 'tx' field")
	}

	// Safely extract body field
	body, ok := tx["body"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid transaction format: missing or invalid 'body' field")
	}

	// Safely extract messages field
	msgs, ok := body["messages"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid transaction format: missing or invalid 'messages' field")
	}

	// Pre-allocate slice with known capacity
	details := make([]map[string]any, 0, len(msgs))

	// Process messages
	for _, msg := range msgs {
		detail, ok := msg.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid message format: expected map[string]any")
		}
		details = append(details, detail)
	}

	return details, nil
}

// GetTxResponse converts a transaction result into a JSON-serializable map and proto transaction.
// It returns an error instead of panicking to allow proper error handling by the caller.
func GetTxResponse(encodingConfig *params.EncodingConfig, idx int, txResult *mq.TxResult, blockResults *mq.BlockResultMsg) (map[string]any, []byte, error) {
	resTx := coretypes.ResultTx{
		Hash:     txResult.Tx.Hash(),
		Height:   blockResults.Height,
		Index:    uint32(idx),
		TxResult: *txResult.ExecTxResults,
		Tx:       txResult.Tx,
	}

	// Create transaction result
	txResponse, err := mkTxResult(encodingConfig.TxConfig, &resTx, blockResults.Timestamp)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create tx result: %w", err)
	}

	// Get proto transaction
	cachedValue := txResponse.Tx.GetCachedValue()
	if cachedValue == nil {
		return nil, nil, fmt.Errorf("cached transaction value is nil")
	}

	protoTx, ok := cachedValue.(*txtypes.Tx)
	if !ok {
		return nil, nil, fmt.Errorf("failed to convert to proto transaction: unexpected type %T", cachedValue)
	}

	// Create and marshal response
	response := &txtypes.GetTxResponse{
		Tx:         protoTx,
		TxResponse: txResponse,
	}

	txResJsonByte, err := codec.ProtoMarshalJSON(response, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal transaction response: %w", err)
	}

	// Unmarshal to map
	var txResJsDict map[string]any
	if err := json.Unmarshal(txResJsonByte, &txResJsDict); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal transaction response: %w", err)
	}

	return txResJsDict, txResJsonByte, nil
}

func ParseTransaction(encodingConfig *params.EncodingConfig, idx int, txResult *mq.TxResult, txResultJsonDict map[string]any, blockResults *mq.BlockResultMsg) (*db.Transaction, error) {
	tx, err := encodingConfig.TxConfig.TxDecoder()(txResult.Tx)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SDK transaction: %w", err)
	}
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return nil, fmt.Errorf("failed to cast SDK transaction to FeeTx: %w", err)
	}
	memoTx, ok := tx.(sdk.TxWithMemo)
	if !ok {
		return nil, fmt.Errorf("failed to cast SDK transaction to TxWithMemo: %w", err)
	}

	var errMsg *string
	if !txResult.ExecTxResults.IsOK() {
		escapedErrMsg := db.NormalizeEscapeString(txResult.ExecTxResults.Log)
		errMsg = &escapedErrMsg
	}

	msgs, err := parseMessages(txResultJsonDict)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message dicts: %w", err)
	}

	messagesJSON, err := json.Marshal(msgs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal messages: %w", err)
	}

	sender, err := parser.GrepSenderFromEvents(txResult.ExecTxResults.Events)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender: %w", err)
	}

	return &db.Transaction{
		ID:          db.GetTxID(txResult.Hash, blockResults.Height),
		Hash:        txResult.Tx.Hash(),
		BlockHeight: blockResults.Height,
		BlockIndex:  idx,
		GasUsed:     txResult.ExecTxResults.GasUsed,
		GasLimit:    int64(feeTx.GetGas()),
		GasFee:      feeTx.GetFee().String(),
		ErrMsg:      errMsg,
		Success:     txResult.ExecTxResults.IsOK(),
		Sender:      sender.String(),
		Memo:        db.NormalizeEscapeString(memoTx.GetMemo()),
		Messages:    messagesJSON,
	}, nil
}
