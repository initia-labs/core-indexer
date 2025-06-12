package flusher

import (
	"fmt"
	"strconv"

	abci "github.com/cometbft/cometbft/abci/types"
	cosmosgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

type ProposalEventProcessor struct {
	newProposals []int32
	TxID         string
}

func newProposalEventProcessor(txID string) *ProposalEventProcessor {
	return &ProposalEventProcessor{
		newProposals: make([]int32, 0),
		TxID:         txID,
	}
}

func (f *Flusher) processProposalEvents(blockResults *mq.BlockResultMsg) error {
	for _, tx := range blockResults.Txs {
		if tx.ExecTxResults.Log == TxParseError {
			continue
		}

		processor := newProposalEventProcessor(tx.Hash)
		// Step 1: Process all events in the transaction
		if err := processor.processTransactionEvents(&tx); err != nil {
			logger.Error().Msgf("Error processing transaction events: %v", err)
			return err
		}

		// Step 3: Update state and database based on processed data
		if err := f.updateStateFromProposalProcessor(processor, blockResults.Height); err != nil {
			return err
		}
	}
	return nil
}

func (f *Flusher) updateStateFromProposalProcessor(processor *ProposalEventProcessor, height int64) error {
	// Update proposals
	for _, proposalID := range processor.newProposals {
		f.stateUpdateManager.proposalsToUpdate[proposalID] = processor.TxID
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
	default:
		return nil
	}
}

// handleEvent routes events to appropriate handlers based on event type
func (p *ProposalEventProcessor) handleSubmitProposalEvent(event abci.Event) error {
	if value, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID); found {
		proposalID, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}
		p.newProposals = append(p.newProposals, int32(proposalID))
	}
	return nil
}
