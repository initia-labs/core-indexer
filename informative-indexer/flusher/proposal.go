package flusher

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	cosmosgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

type ProposalEventProcessor struct {
	newProposals          []int32
	proposalStatusChanges map[int32]db.ProposalStatus
	TxID                  string
	height                int32
}

func newProposalEventProcessor(txID string, height int64) *ProposalEventProcessor {
	return &ProposalEventProcessor{
		newProposals:          make([]int32, 0),
		proposalStatusChanges: make(map[int32]db.ProposalStatus),
		TxID:                  txID,
		height:                int32(height),
	}
}

func (f *Flusher) processProposalEvents(txResult *mq.TxResult, height int64, _ *db.Transaction) error {
	processor := newProposalEventProcessor(db.GetTxID(txResult.Hash, height), height)
	// Step 1: Process all events in the transaction
	if err := processor.processTransactionEvents(txResult); err != nil {
		logger.Error().Msgf("Error processing transaction events: %v", err)
		return err
	}

	// Step 3: Update state and database based on processed data
	if err := f.updateStateFromProposalProcessor(processor, height); err != nil {
		return err
	}
	return nil
}

func (f *Flusher) updateStateFromProposalProcessor(processor *ProposalEventProcessor, height int64) error {
	// Update proposals
	for _, proposalID := range processor.newProposals {
		f.stateUpdateManager.proposalsToUpdate[proposalID] = processor.TxID
	}

	for proposalID, newStatus := range processor.proposalStatusChanges {
		proposal := db.Proposal{ID: proposalID, Status: string(newStatus)}
		if isProposalResolved(newStatus) {
			proposal.ResolvedHeight = &processor.height
		}

		f.stateUpdateManager.proposalStatusChanges[proposalID] = proposal
	}

	return nil
}

func (p *ProposalEventProcessor) processTransactionEvents(tx *mq.TxResult) error {
	for _, event := range tx.ExecTxResults.Events {
		if err := p.handleEvent(event); err != nil {
			return fmt.Errorf("failed to handle event %s: %w", event.Type, err)
		}
	}
	return nil
}

// handleEvent routes events to appropriate handlers based on event type
func (p *ProposalEventProcessor) handleEvent(event abci.Event) error {
	switch event.Type {
	case cosmosgovtypes.EventTypeSubmitProposal:
		return p.handleSubmitProposalEvent(event)
	case cosmosgovtypes.EventTypeProposalDeposit:
		return p.handleProposalDepositEvent(event)
	case cosmosgovtypes.EventTypeCancelProposal:
		return p.handleCancelProposalEvent(event)
	default:
		return nil
	}
}

// handleEvent routes events to appropriate handlers based on event type
func (p *ProposalEventProcessor) handleSubmitProposalEvent(event abci.Event) error {
	if value, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID); found {
		proposalID, err := parser.ParseInt32(value)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}
		p.newProposals = append(p.newProposals, proposalID)
	}
	return nil
}

func (p *ProposalEventProcessor) handleProposalDepositEvent(event abci.Event) error {
	if value, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyVotingPeriodStart); found {
		proposalID, err := parser.ParseInt32(value)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}
		p.proposalStatusChanges[proposalID] = db.ProposalStatusVotingPeriod
	}
	return nil
}

func (p *ProposalEventProcessor) handleCancelProposalEvent(event abci.Event) error {
	if value, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID); found {
		proposalID, err := parser.ParseInt32(value)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}
		p.proposalStatusChanges[proposalID] = db.ProposalStatusCancelled
	}
	return nil
}

// EndBlock

type ProposalEndBlockEventProcessor struct {
	proposalStatusChanges    map[int32]db.ProposalStatus
	proposalExpeditedChanges map[int32]bool
	height                   int32
}

func newProposalEndBlockEventProcessor(height int64) *ProposalEndBlockEventProcessor {
	return &ProposalEndBlockEventProcessor{
		proposalStatusChanges:    make(map[int32]db.ProposalStatus),
		proposalExpeditedChanges: make(map[int32]bool),
		height:                   int32(height),
	}
}

func (f *Flusher) processProposalEndBlockEvents(blockResults *mq.BlockResultMsg) error {
	processor := newProposalEndBlockEventProcessor(blockResults.Height)
	// Step 1: Process all events in the transaction
	if err := processor.processEndBlockEvents(blockResults); err != nil {
		logger.Error().Msgf("Error processing transaction events: %v", err)
		return err
	}

	// Step 3: Update state and database based on processed data
	if err := f.updateStateFromProposalEndBlockProcessor(processor); err != nil {
		return err
	}
	return nil
}

func (f *Flusher) updateStateFromProposalEndBlockProcessor(processor *ProposalEndBlockEventProcessor) error {
	// Update proposals
	for proposalID, newStatus := range processor.proposalStatusChanges {
		proposal := db.Proposal{ID: proposalID, Status: string(newStatus)}
		if isProposalResolved(newStatus) {
			proposal.ResolvedHeight = &processor.height
		}
		f.stateUpdateManager.proposalStatusChanges[proposalID] = proposal
	}

	for proposalID := range processor.proposalExpeditedChanges {
		f.dbBatchInsert.proposalExpeditedChanges[proposalID] = true
	}

	return nil
}

func (p *ProposalEndBlockEventProcessor) processEndBlockEvents(blockResults *mq.BlockResultMsg) error {
	for _, event := range blockResults.FinalizeBlockEvents {
		if err := p.handleEndBlockEvent(event); err != nil {
			return fmt.Errorf("failed to handle event %s: %w", event.Type, err)
		}
	}
	return nil
}

// handleEvent routes events to appropriate handlers based on event type
func (p *ProposalEndBlockEventProcessor) handleEndBlockEvent(event abci.Event) error {
	switch event.Type {
	case cosmosgovtypes.EventTypeInactiveProposal:
		return p.handleProposalEndblockEvent(event)
	case cosmosgovtypes.EventTypeActiveProposal:
		return p.handleProposalEndblockEvent(event)
	default:
		return nil
	}
}

func (p *ProposalEndBlockEventProcessor) handleProposalEndblockEvent(event abci.Event) error {
	if value, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID); found {
		proposalID, err := parser.ParseInt32(value)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}

		if value, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalResult); found {
			if isExpeditedRejected(value) {
				p.proposalExpeditedChanges[proposalID] = true
			} else {
				result, err := parseProposalEndBlockAttributeValue(value)
				if err != nil {
					return fmt.Errorf("failed to parse proposal result: %w", err)
				}
				p.proposalStatusChanges[proposalID] = result
			}
		}
	}
	return nil
}
