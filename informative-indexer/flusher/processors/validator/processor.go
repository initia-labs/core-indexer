package validator

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/initia/app/params"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	"github.com/initia-labs/core-indexer/informative-indexer/flusher/processors"
	statetracker "github.com/initia-labs/core-indexer/informative-indexer/flusher/state-tracker"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

var _ processors.Processor = &Processor{}

func (p *Processor) InitProcessor(height int64, validatorMap map[string]mstakingtypes.Validator) {
	p.height = height
	p.validatorMap = validatorMap
	p.stakeChanges = make([]db.ValidatorBondedTokenChange, 0)
	p.validators = make(map[string]bool)
	p.slashEvents = make([]db.ValidatorSlashEvent, 0)

	p.txProcessor = nil
}

func (p *Processor) Name() string {
	return "validator"
}

func (p *Processor) ProcessBeginBlockEvents(finalizeBlockEvents *[]abci.Event) error {
	for _, event := range *finalizeBlockEvents {
		p.handleBeginBlockEvent(event)
	}
	return nil
}

func (p *Processor) ProcessEndBlockEvents(finalizeBlockEvents *[]abci.Event) error {
	return nil
}

// TODO: or split into 4 fn interfaces???
func (p *Processor) ProcessTransactions(tx *mq.TxResult, encodingConfig *params.EncodingConfig) error {
	p.newTxProcessor(tx.Hash)

	if err := p.processSDKMessages(tx, encodingConfig); err != nil {
		return fmt.Errorf("failed to process sdk message: %w", err)
	}

	if err := p.processTransactionEvents(tx); err != nil {
		return fmt.Errorf("failed to process tx events: %w", err)
	}

	if err := p.resolveTxProcessor(); err != nil {
		return fmt.Errorf("failed to resolve tx: %w", err)
	}

	return nil
}

func (p *Processor) TrackState(stateUpdateManager *statetracker.StateUpdateManager, dbBatchInsert *statetracker.DBBatchInsert) error {
	for addr := range p.validators {
		stateUpdateManager.Validators[addr] = true
	}
	dbBatchInsert.AddValidatorBondedTokenTxs(p.stakeChanges...)
	dbBatchInsert.AddValidatorSlashEvents(p.slashEvents...)

	return nil
}
