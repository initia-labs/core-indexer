package proposal

import (
	"fmt"
	"maps"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/app/params"
	vmapi "github.com/initia-labs/movevm/api"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/processors"
	statetracker "github.com/initia-labs/core-indexer/informative-indexer/indexer/state-tracker"
	"github.com/initia-labs/core-indexer/informative-indexer/indexer/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

var _ processors.Processor = &Processor{}

func (p *Processor) InitProcessor(height int64, validatorMap map[string]db.ValidatorAddress) {
	p.height = height
	p.int32Height = int32(height)
	p.validatorMap = validatorMap
	p.newProposals = make(map[int32]string)
	p.proposalStatusChanges = make(map[int32]db.ProposalStatus)
	p.proposalDeposits = make([]db.ProposalDeposit, 0)
	p.totalDepositChanges = make(map[int32][]sdk.Coin)
	p.proposalVotes = make([]db.ProposalVote, 0)
	p.proposalExpeditedChanges = make(map[int32]bool)
	p.proposalEmergencyNextTally = make(map[int32]*time.Time)
	p.modulePublishedEvents = make([]db.ModuleHistory, 0)
	p.moduleProposals = make([]db.ModuleProposal, 0)
	p.newModules = make(map[vmapi.ModuleInfoResponse]bool)

	p.txProcessor = nil
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
			return fmt.Errorf("failed to handle end block event %s: %w", event.Type, err)
		}
	}
	return nil
}

func (p *Processor) NewTxProcessor(txData *db.Transaction) {
	p.txProcessor = &TxProcessor{
		txData: txData,
	}
}

func (p *Processor) ProcessSDKMessages(tx *mq.TxResult, encodingConfig *params.EncodingConfig) error {
	return nil
}

func (p *Processor) ProcessTransactionEvents(tx *mq.TxResult) error {
	for _, event := range tx.ExecTxResults.Events {
		if err := p.handleEvent(event); err != nil {
			return fmt.Errorf("failed to handle tx event %s: %w", event.Type, err)
		}
	}
	return nil
}

func (p *Processor) ResolveTxProcessor() error {
	return nil
}

func (p *Processor) TrackState(stateUpdateManager *statetracker.StateUpdateManager, dbBatchInsert *statetracker.DBBatchInsert) error {
	// Update proposals
	maps.Copy(stateUpdateManager.ProposalsToUpdate, p.newProposals)

	dbBatchInsert.ProposalDeposits = p.proposalDeposits
	dbBatchInsert.TotalDepositChanges = p.totalDepositChanges

	dbBatchInsert.ProposalVotes = p.proposalVotes

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
