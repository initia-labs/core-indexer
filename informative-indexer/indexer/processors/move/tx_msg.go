package move

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"
	vmtypes "github.com/initia-labs/movevm/types"
)

func (p *Processor) handleMsg(msg sdk.Msg, isTxOk bool) error {
	switch msg := msg.(type) {
	case *movetypes.MsgExecute:
		p.txProcessor.txData.IsMoveExecute = true
		if err := p.handleMoveExecuteEventIsEntry(msg.ModuleAddress, msg.ModuleName); err != nil && isTxOk {
			return fmt.Errorf("failed to process MsgExecute: %w", err)
		}
		return nil
	case *movetypes.MsgExecuteJSON:
		p.txProcessor.txData.IsMoveExecute = true
		if err := p.handleMoveExecuteEventIsEntry(msg.ModuleAddress, msg.ModuleName); err != nil && isTxOk {
			return fmt.Errorf("failed to process MsgExecuteJSON: %w", err)
		}
		return nil
	case *movetypes.MsgScript:
		p.txProcessor.txData.IsMoveScript = true
		return nil
	default:
		return nil
	}
}

// handleMoveExecuteEventIsEntry processes entry point execution events,
// marking modules as entry points when they are executed directly
func (p *Processor) handleMoveExecuteEventIsEntry(moduleAddress, moduleName string) error {
	vmAddr, err := vmtypes.NewAccountAddress(moduleAddress)
	if err != nil {
		return fmt.Errorf("invalid module address: %w", err)
	}
	p.txProcessor.modulesInTx[vmapi.ModuleInfoResponse{Address: vmAddr, Name: moduleName}] = true
	return nil
}
