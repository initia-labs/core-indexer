package mq

import (
	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"testing"
)

func TestBlockResultMessageRejectsMalformedData(t *testing.T) {
	block := &coretypes.ResultBlock{Block: &types.Block{}}
	for _, tc := range []struct {
		name    string
		block   *coretypes.ResultBlock
		results *coretypes.ResultBlockResults
	}{
		{"missing response", nil, &coretypes.ResultBlockResults{}},
		{"missing block", &coretypes.ResultBlock{}, &coretypes.ResultBlockResults{}},
		{"missing results", block, nil},
		{"excess transaction results", block, &coretypes.ResultBlockResults{TxsResults: []*abci.ExecTxResult{{}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewBlockResultMsgBytes(tc.block, tc.results); err == nil {
				t.Fatal("expected malformed data error")
			}
		})
	}
}
