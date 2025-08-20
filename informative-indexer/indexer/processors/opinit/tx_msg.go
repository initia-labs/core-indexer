package opinit

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

func (p *Processor) handleMsg(msg sdk.Msg) {
	switch msg := msg.(type) {
	case *ophosttypes.MsgRecordBatch:
		p.txProcessor.txData.IsOpinit = true
		p.txProcessor.relatedBridge[msg.BridgeId] = true
	case *ophosttypes.MsgCreateBridge:
		p.txProcessor.txData.IsOpinit = true
	case *ophosttypes.MsgProposeOutput:
		p.txProcessor.txData.IsOpinit = true
		p.txProcessor.relatedBridge[msg.BridgeId] = true
	case *ophosttypes.MsgDeleteOutput:
		p.txProcessor.txData.IsOpinit = true
		p.txProcessor.relatedBridge[msg.BridgeId] = true
	case *ophosttypes.MsgInitiateTokenDeposit:
		p.txProcessor.txData.IsOpinit = true
		p.txProcessor.relatedBridge[msg.BridgeId] = true
	case *ophosttypes.MsgFinalizeTokenWithdrawal:
		p.txProcessor.txData.IsOpinit = true
		p.txProcessor.relatedBridge[msg.BridgeId] = true
	case *ophosttypes.MsgUpdateProposer:
		p.txProcessor.txData.IsOpinit = true
		p.txProcessor.relatedBridge[msg.BridgeId] = true
	case *ophosttypes.MsgUpdateChallenger:
		p.txProcessor.txData.IsOpinit = true
		p.txProcessor.relatedBridge[msg.BridgeId] = true
	case *ophosttypes.MsgUpdateOracleConfig:
		p.txProcessor.txData.IsOpinit = true
		p.txProcessor.relatedBridge[msg.BridgeId] = true
	case *ophosttypes.MsgUpdateBatchInfo:
		p.txProcessor.txData.IsOpinit = true
		p.txProcessor.relatedBridge[msg.BridgeId] = true
	case *ophosttypes.MsgUpdateMetadata:
		p.txProcessor.txData.IsOpinit = true
		p.txProcessor.relatedBridge[msg.BridgeId] = true
	case *ophosttypes.MsgUpdateParams:
		p.txProcessor.txData.IsOpinit = true
	}
}
