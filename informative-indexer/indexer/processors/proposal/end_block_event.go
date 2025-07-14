package proposal

import (
	"fmt"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cosmosgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	movetypes "github.com/initia-labs/initia/x/move/types"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/types"
	"github.com/initia-labs/core-indexer/informative-indexer/indexer/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/parser"
	govtypes "github.com/initia-labs/initia/x/gov/types"
)

// handleEvent routes events to appropriate handlers based on event type
func (p *Processor) handleEndBlockEvent(event abci.Event) error {
	switch event.Type {
	case cosmosgovtypes.EventTypeInactiveProposal:
		return p.handleProposalEndblockEvent(event)
	case cosmosgovtypes.EventTypeActiveProposal:
		return p.handleProposalEndblockEvent(event)
	case govtypes.EventTypeEmergencyProposal:
		return p.handleEmergencyProposalEvent(event)
	case movetypes.EventTypeMove:
		return p.handleMoveEvent(event)
	default:
		return nil
	}
}

func (p *Processor) handleProposalEndblockEvent(event abci.Event) error {
	if value, found := utils.FindAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID); found {
		proposalID, err := parser.ParseInt32(value)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}

		if value, found := utils.FindAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalResult); found {
			if utils.IsExpeditedRejected(value) {
				p.proposalExpeditedChanges[proposalID] = true
			} else {
				result, err := utils.ParseProposalEndBlockAttributeValue(value)
				if err != nil {
					return fmt.Errorf("failed to parse proposal result: %w", err)
				}
				// TODO: refactor this
				for idx := len(p.modulePublishedEvents) - 1; idx >= 0; idx-- {
					if p.modulePublishedEvents[idx].ProposalID == nil {
						p.modulePublishedEvents[idx].ProposalID = &proposalID
						p.moduleProposals = append(p.moduleProposals, db.ModuleProposal{
							ProposalID: proposalID,
							ModuleID:   p.modulePublishedEvents[idx].ModuleID,
						})
					} else {
						break
					}
				}
				p.proposalStatusChanges[proposalID] = result
			}
		}
	}
	return nil
}

func (p *Processor) handleEmergencyProposalEvent(event abci.Event) error {
	if value, found := utils.FindAttribute(event.Attributes, govtypes.AttributeKeyProposalID); found {
		proposalID, err := parser.ParseInt32(value)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}

		// TODO: bump initia version and replace with `govtypes.AttributeKeyNextTallyTime`
		if value, found := utils.FindAttribute(event.Attributes, "next_tally_time"); found {
			nextTallyTime, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return fmt.Errorf("failed to parse emergency next tally time: %w", err)
			}
			p.proposalEmergencyNextTally[proposalID] = &nextTallyTime
		}
	}
	return nil
}

func (p *Processor) handleMoveEvent(event abci.Event) error {
	if value, found := utils.FindAttribute(event.Attributes, movetypes.AttributeKeyTypeTag); found {
		switch value {
		case types.ModulePublishedEventKey:
			return p.handlePublishEvent(event)
		}
	}
	return nil
}

// handlePublishEvent processes module publish events, recording new modules
func (p *Processor) handlePublishEvent(event abci.Event) error {
	if value, found := utils.FindAttribute(event.Attributes, movetypes.AttributeKeyData); found {
		module, upgradePolicy, err := parser.DecodePublishModuleData(value)
		if err != nil {
			return fmt.Errorf("failed to decode publish module data: %w", err)
		}
		p.newModules[module] = true
		p.modulePublishedEvents = append(p.modulePublishedEvents, db.ModuleHistory{
			ModuleID:      db.GetModuleID(module),
			Remark:        db.JSON("{}"),
			BlockHeight:   p.int32Height,
			UpgradePolicy: db.GetUpgradePolicy(upgradePolicy),
		})
	}
	return nil
}
