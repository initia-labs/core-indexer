package flusher

import (
	"fmt"
	"strconv"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmosgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

type ProposalEventProcessor struct {
	newProposals     []int32
	TxID             string
	proposalDeposits []db.ProposalDeposit
}

func newProposalEventProcessor(txID string) *ProposalEventProcessor {
	return &ProposalEventProcessor{
		newProposals:     make([]int32, 0),
		TxID:             txID,
		proposalDeposits: make([]db.ProposalDeposit, 0),
	}
}

func (f *Flusher) processProposalEvents(txResult *mq.TxResult, height int64, _ *db.Transaction) error {
	processor := newProposalEventProcessor(db.GetTxID(txResult.Hash, height))
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

	f.dbBatchInsert.proposalDeposits = append(f.dbBatchInsert.proposalDeposits, processor.proposalDeposits...)
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
	// case cosmosgovtypes.EventTypeProposalVote:
	// 	return p.handleProposalVoteEvent(event)
	default:
		return nil
	}
}

// handleEvent routes events to appropriate handlers based on event type
func (p *ProposalEventProcessor) handleSubmitProposalEvent(event abci.Event) error {
	if value, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID); found {
		proposalIDInt, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}
		p.newProposals = append(p.newProposals, int32(proposalIDInt))
	}
	return nil
}

func (p *ProposalEventProcessor) handleProposalDepositEvent(event abci.Event) error {
	proposalID, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID)
	if !found {
		return fmt.Errorf("failed to filter proposal id")
	}

	depositor, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyDepositor)
	if !found {
		return fmt.Errorf("failed to filter depositor")
	}

	coins, found := findAttribute(event.Attributes, sdk.AttributeKeyAmount)
	if !found {
		return fmt.Errorf("failed to filter amount")
	}

	amount, denom, err := parser.ParseCoinAmount(coins)
	if err != nil {
		return fmt.Errorf("failed to parse amount: %w", err)
	}

	proposalIDInt, err := strconv.ParseInt(proposalID, 10, 32)
	if err != nil {
		return fmt.Errorf("failed to parse proposal id: %w", err)
	}

	p.proposalDeposits = append(p.proposalDeposits, db.ProposalDeposit{
		Depositor:     depositor,
		Amount:        db.JSON(fmt.Sprintf(`[{"amount": "%d", "denom": "%s"}]`, amount, denom)),
		ProposalID:    int32(proposalIDInt),
		TransactionID: p.TxID,
	})
	return nil
}

// func (p *ProposalEventProcessor) handleProposalVoteEvent(event abci.Event) error {
// 	proposalID, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID)
// 	if !found {
// 		return fmt.Errorf("failed to filter proposal id")
// 	}

// 	proposalIDInt, err := strconv.ParseInt(proposalID, 10, 32)
// 	if err != nil {
// 		return fmt.Errorf("failed to parse proposal id: %w", err)
// 	}

// 	voter, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyVoter)
// 	if !found {
// 		return fmt.Errorf("failed to filter voter")
// 	}

// 	rawOptions, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyOption)
// 	if !found {
// 		return fmt.Errorf("failed to filter option")
// 	}

// 	options, err := parser.DecodeEvent[govv1types.WeightedVoteOptions](rawOptions)
// 	if err != nil {
// 		return fmt.Errorf("failed to decode option: %w", err)
// 	}
// 	votes := map[govv1types.VoteOption]string{govv1types.OptionYes: "0", govv1types.OptionAbstain: "0", govv1types.OptionNo: "0", govv1types.OptionNoWithVeto: "0"}
// 	for _, option := range options {
// 		votes[option.GetOption()] = option.GetWeight()
// 	}
// 	_ = voter
// 	_ = proposalIDInt
// 	_ = rawOptions
// 	return nil
// }
