package flusher

import (
	"fmt"
	"strconv"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmosgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmapi "github.com/initia-labs/movevm/api"

	"github.com/initia-labs/core-indexer/informative-indexer/flusher/types"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
	govtypes "github.com/initia-labs/initia/x/gov/types"
)

type ProposalEventProcessor struct {
	newProposals          []int32
	proposalStatusChanges map[int32]db.ProposalStatus
	TxID                  string
	height                int32
	proposalDeposits      []db.ProposalDeposit
	proposalVotes         []db.ProposalVote
}

func newProposalEventProcessor(txID string, height int64) *ProposalEventProcessor {
	return &ProposalEventProcessor{
		newProposals:          make([]int32, 0),
		proposalStatusChanges: make(map[int32]db.ProposalStatus),
		TxID:                  txID,
		height:                int32(height),
		proposalDeposits:      make([]db.ProposalDeposit, 0),
		proposalVotes:         make([]db.ProposalVote, 0),
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
	if err := f.updateStateFromProposalProcessor(processor); err != nil {
		return err
	}
	return nil
}

func (f *Flusher) updateStateFromProposalProcessor(processor *ProposalEventProcessor) error {
	// Update proposals
	for _, proposalID := range processor.newProposals {
		f.stateUpdateManager.proposalsToUpdate[proposalID] = processor.TxID
	}

	f.dbBatchInsert.proposalDeposits = append(f.dbBatchInsert.proposalDeposits, processor.proposalDeposits...)
	f.dbBatchInsert.proposalVotes = append(f.dbBatchInsert.proposalVotes, processor.proposalVotes...)

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
	case cosmosgovtypes.EventTypeProposalVote:
		return p.handleProposalVoteEvent(event)
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

	value, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID)
	if !found {
		return fmt.Errorf("failed to filter proposal id")
	}

	proposalID, err := parser.ParseInt32(value)
	if err != nil {
		return fmt.Errorf("failed to parse proposal id: %w", err)
	}

	depositor, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyDepositor)
	if !found {
		return fmt.Errorf("failed to filter depositor")
	}

	coin, found := findAttribute(event.Attributes, sdk.AttributeKeyAmount)
	if !found {
		return fmt.Errorf("failed to filter amount")
	}

	amount, denom, err := parser.ParseCoinAmount(coin)
	if err != nil {
		return fmt.Errorf("failed to parse amount: %w", err)
	}

	p.proposalDeposits = append(p.proposalDeposits, db.ProposalDeposit{
		Depositor:     depositor,
		Amount:        db.JSON(fmt.Sprintf(`[{"amount": "%d", "denom": "%s"}]`, amount, denom)),
		ProposalID:    proposalID,
		TransactionID: p.TxID,
	})
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
	proposalStatusChanges      map[int32]db.ProposalStatus
	proposalExpeditedChanges   map[int32]bool
	modulePublishedEvents      []db.ModuleHistory
	latestProposalID           int32
	newModules                 map[vmapi.ModuleInfoResponse]bool
	height                     int32
	proposalEmergencyNextTally map[int32]*time.Time
}

func newProposalEndBlockEventProcessor(height int64) *ProposalEndBlockEventProcessor {
	return &ProposalEndBlockEventProcessor{
		proposalStatusChanges:      make(map[int32]db.ProposalStatus),
		proposalExpeditedChanges:   make(map[int32]bool),
		modulePublishedEvents:      make([]db.ModuleHistory, 0),
		latestProposalID:           0,
		newModules:                 make(map[vmapi.ModuleInfoResponse]bool),
		height:                     int32(height),
		proposalEmergencyNextTally: make(map[int32]*time.Time),
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

	f.dbBatchInsert.modulePublishedEvents = append(f.dbBatchInsert.modulePublishedEvents, processor.modulePublishedEvents...)

	for module := range processor.newModules {
		f.stateUpdateManager.modules[module] = nil
	}

	for proposalID, nextTallyTime := range processor.proposalEmergencyNextTally {
		f.dbBatchInsert.proposalEmergencyNextTally[proposalID] = nextTallyTime
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
	case movetypes.EventTypeMove:
		return p.handleMoveEvent(event)
	case govtypes.EventTypeEmergencyProposal:
		return p.handleEmergencyProposalEvent(event)
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
				// TODO: refactor this
				for idx := len(p.modulePublishedEvents) - 1; idx >= 0; idx-- {
					if p.modulePublishedEvents[idx].ProposalID == nil {
						p.modulePublishedEvents[idx].ProposalID = &proposalID
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

func (p *ProposalEventProcessor) handleProposalVoteEvent(event abci.Event) error {
	value, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyProposalID)
	if !found {
		return fmt.Errorf("failed to filter proposal id")
	}

	proposalID, err := parser.ParseInt32(value)
	if err != nil {
		return fmt.Errorf("failed to parse proposal id: %w", err)
	}

	voter, found := findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyVoter)
	if !found {
		return fmt.Errorf("failed to filter voter")
	}

	value, found = findAttribute(event.Attributes, cosmosgovtypes.AttributeKeyOption)
	if !found {
		return fmt.Errorf("failed to filter option")
	}

	options, err := parser.DecodeEvent[govv1types.WeightedVoteOptions](value)
	if err != nil {
		return fmt.Errorf("failed to decode option: %w", err)
	}

	votes := map[govv1types.VoteOption]string{
		govv1types.OptionYes:        "0",
		govv1types.OptionAbstain:    "0",
		govv1types.OptionNo:         "0",
		govv1types.OptionNoWithVeto: "0",
	}
	for _, option := range options {
		votes[option.GetOption()] = option.GetWeight()
	}

	yesVote, _ := strconv.ParseFloat(votes[govv1types.OptionYes], 64)
	noVote, _ := strconv.ParseFloat(votes[govv1types.OptionNo], 64)
	abstainVote, _ := strconv.ParseFloat(votes[govv1types.OptionAbstain], 64)
	noWithVetoVote, _ := strconv.ParseFloat(votes[govv1types.OptionNoWithVeto], 64)

	p.proposalVotes = append(p.proposalVotes, db.ProposalVote{
		Voter:          voter,
		ProposalID:     proposalID,
		TransactionID:  p.TxID,
		IsVoteWeighted: len(options) > 1,
		IsValidator:    false,
		Yes:            yesVote,
		No:             noVote,
		Abstain:        abstainVote,
		NoWithVeto:     noWithVetoVote,
	})
	return nil
}

func (p *ProposalEndBlockEventProcessor) handleMoveEvent(event abci.Event) error {
	if value, found := findAttribute(event.Attributes, movetypes.AttributeKeyTypeTag); found {
		switch value {
		case types.ModulePublishedEventKey:
			return p.handlePublishEvent(event)
		}
	}
	return nil
}

// handlePublishEvent processes module publish events, recording new modules
func (p *ProposalEndBlockEventProcessor) handlePublishEvent(event abci.Event) error {
	if value, found := findAttribute(event.Attributes, movetypes.AttributeKeyData); found {
		module, upgradePolicy, err := parser.DecodePublishModuleData(value)
		if err != nil {
			return fmt.Errorf("failed to decode publish module data: %w", err)
		}
		p.newModules[module] = true
		p.modulePublishedEvents = append(p.modulePublishedEvents, db.ModuleHistory{
			ModuleID:      db.GetModuleID(module),
			Remark:        db.JSON("{}"),
			BlockHeight:   p.height,
			UpgradePolicy: db.GetUpgradePolicy(upgradePolicy),
		})
	}
	return nil
}

func (p *ProposalEndBlockEventProcessor) handleEmergencyProposalEvent(event abci.Event) error {
	if value, found := findAttribute(event.Attributes, govtypes.AttributeKeyProposalID); found {
		proposalID, err := parser.ParseInt32(value)
		if err != nil {
			return fmt.Errorf("failed to parse proposal id: %w", err)
		}

		// TODO: bump initia version and replace with `govtypes.AttributeKeyNextTallyTime`
		if value, found := findAttribute(event.Attributes, "next_tally_time"); found {
			nextTallyTime, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return fmt.Errorf("failed to parse emergency next tally time: %w", err)
			}
			p.proposalEmergencyNextTally[proposalID] = &nextTallyTime
		}
	}
	return nil
}
