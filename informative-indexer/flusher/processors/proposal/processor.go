package proposal

import (
	"fmt"
	"maps"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/initia/app/params"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	vmapi "github.com/initia-labs/movevm/api"

	statetracker "github.com/initia-labs/core-indexer/informative-indexer/flusher/state-tracker"
	"github.com/initia-labs/core-indexer/informative-indexer/flusher/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

func (p *Processor) InitProcessor(height int64, validatorMap map[string]mstakingtypes.Validator) {
	p.height = height
	p.int32Height = int32(height)
	p.validatorMap = validatorMap
	p.newProposals = make(map[int32]string)
	p.proposalStatusChanges = make(map[int32]db.ProposalStatus)
	p.proposalDeposits = make([]db.ProposalDeposit, 0)
	p.proposalVotes = make([]db.ProposalVote, 0)
	p.proposalExpeditedChanges = make(map[int32]bool)
	p.proposalEmergencyNextTally = make(map[int32]*time.Time)
	p.modulePublishedEvents = make([]db.ModuleHistory, 0)
	p.moduleProposals = make([]db.ModuleProposal, 0)
	p.newModules = make(map[vmapi.ModuleInfoResponse]bool)
}

func (p *Processor) Name() string {
	return "proposal"
}

func (p *Processor) ProcessBeginBlockEvents(finalizeBlockEvents *[]abci.Event) error {
	return nil
}

func (p *Processor) ProcessEndBlockEvents(finalizeBlockEvents *[]abci.Event) error {
	for _, event := range *finalizeBlockEvents {
		if err := p.handleEndBlockEvent(event); err != nil {
			return fmt.Errorf("failed to handle event %s: %w", event.Type, err)
		}
	}
	return nil
}

func (p *Processor) ProcessTransactions(tx *mq.TxResult, encodingConfig *params.EncodingConfig) error {
	p.newTxProcessor(tx.Hash)

	if err := p.processTransactionEvents(tx); err != nil {
		return fmt.Errorf("failed to process tx events: %w", err)
	}
	return nil
}

func (p *Processor) TrackState(stateUpdateManager *statetracker.StateUpdateManager, dbBatchInsert *statetracker.DBBatchInsert) error {
	// Update proposals
	maps.Copy(stateUpdateManager.ProposalsToUpdate, p.newProposals)

	dbBatchInsert.ProposalDeposits = append(dbBatchInsert.ProposalDeposits, p.proposalDeposits...)
	dbBatchInsert.ProposalVotes = append(dbBatchInsert.ProposalVotes, p.proposalVotes...)

	for proposalID, newStatus := range p.proposalStatusChanges {
		proposal := db.Proposal{ID: proposalID, Status: string(newStatus)}
		if utils.IsProposalResolved(newStatus) {
			proposal.ResolvedHeight = &p.int32Height
		}

		stateUpdateManager.ProposalStatusChanges[proposalID] = proposal
	}

	for proposalID := range p.proposalExpeditedChanges {
		dbBatchInsert.ProposalExpeditedChanges[proposalID] = true
	}

	maps.Copy(dbBatchInsert.ProposalEmergencyNextTally, p.proposalEmergencyNextTally)

	dbBatchInsert.ModulePublishedEvents = append(dbBatchInsert.ModulePublishedEvents, p.modulePublishedEvents...)

	for module := range p.newModules {
		stateUpdateManager.Modules[module] = nil
	}

	return nil
}
