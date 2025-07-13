package validator

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"

	"github.com/initia-labs/core-indexer/informative-indexer/flusher/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
)

// handleEvent routes events to appropriate handlers based on event type
func (p *Processor) handleBeginBlockEvent(event abci.Event) error {
	switch event.Type {
	case slashingtypes.EventTypeSlash:
		return p.handleSlashEvent(event)
	default:
		return nil
	}
}

func (p *Processor) handleSlashEvent(event abci.Event) error {
	if value, found := utils.FindAttribute(event.Attributes, slashingtypes.AttributeKeyJailed); found {
		// TODO: is the jailed validator gonna be in the validator map?
		validator, ok := p.validatorMap[value]
		if !ok {
			return fmt.Errorf("failed to map validator address: %s", value)
		}

		p.validators[validator.OperatorAddress] = true
		p.slashEvents = append(p.slashEvents, db.ValidatorSlashEvent{
			ValidatorAddress: validator.OperatorAddress,
			BlockHeight:      p.height,
			Type:             string(db.Jailed),
		})
	}
	return nil
}
