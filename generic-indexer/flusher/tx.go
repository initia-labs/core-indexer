package flusher

import (
	"encoding/json"
	"fmt"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	ctypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/initia-labs/core-indexer/generic-indexer/common"
)

type intoAny interface {
	AsAny() *codectypes.Any
}

func parseTxMessages(messages []types.Msg, md []common.JsDict) []common.JsDict {
	var parsedMessages []common.JsDict
	for idx, msg := range messages {
		parsedMessages = append(parsedMessages, common.JsDict{
			"type":   types.MsgTypeURL(msg),
			"detail": md[idx],
		})
	}

	return parsedMessages
}

// MkTxResult returns a sdk.TxResponse from the given Tendermint ResultTx.
func MkTxResult(txConfig client.TxConfig, resTx *coretypes.ResultTx, blockTime time.Time) (*types.TxResponse, error) {
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

func (f *Flusher) getTxResponse(blockTime time.Time, txHash ctypes.Tx, resTx coretypes.ResultTx) (common.JsDict, []byte, txtypes.Tx) {
	txResult, err := MkTxResult(f.encodingConfig.TxConfig, &resTx, blockTime)
	if err != nil {
		panic(err)
	}
	protoTx, ok := txResult.Tx.GetCachedValue().(*txtypes.Tx)
	if !ok {
		panic("cannot make proto tx")
	}
	txResJson, err := codec.ProtoMarshalJSON(&txtypes.GetTxResponse{
		Tx:         protoTx,
		TxResponse: txResult,
	}, nil)
	if err != nil {
		panic(err)
	}
	var txResJsDict common.JsDict
	err = json.Unmarshal(txResJson, &txResJsDict)
	if err != nil {
		panic(err)
	}
	return txResJsDict, txResJson, *protoTx
}

// getMessageDicts returns an array of JsDict decoded version for messages in the provided transaction.
func getMessageDicts(txResJsDict common.JsDict) []common.JsDict {
	details := make([]common.JsDict, 0)
	tx := txResJsDict["tx"].(map[string]interface{})
	body := tx["body"].(map[string]interface{})
	msgs := body["messages"].([]interface{})
	for _, msg := range msgs {
		detail := msg.(map[string]interface{})
		details = append(details, detail)
	}
	return details
}
