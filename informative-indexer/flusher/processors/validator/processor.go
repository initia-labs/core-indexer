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
		if err := p.handleBeginBlockEvent(event); err != nil {
			return fmt.Errorf("failed to handle begin block event %s: %w", event.Type, err)
		}
	}
	return nil
}

func (p *Processor) ProcessEndBlockEvents(finalizeBlockEvents *[]abci.Event) error {
	return nil
}

func (p *Processor) NewTxProcessor(txData *db.Transaction) {
	p.txProcessor = &TxProcessor{
		txData:         txData,
		txStakeChanges: make(map[string]int64),
	}
}

// ProcessSDKMessages processes SDK transaction messages to identify entry points
func (p *Processor) ProcessSDKMessages(tx *mq.TxResult, encodingConfig *params.EncodingConfig) error {
	if !tx.ExecTxResults.IsOK() {
		return nil
	}

	sdkTx, err := encodingConfig.TxConfig.TxDecoder()(tx.Tx)
	if err != nil {
		return fmt.Errorf("failed to decode SDK transaction: %w", err)
	}

	for _, msg := range sdkTx.GetMsgs() {
		p.handleMsgs(msg)
	}

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
	processedStakeChanges, err := processStakeChanges(p.txProcessor.txStakeChanges, p.txProcessor.txData.ID, p.height)
	if err != nil {
		return fmt.Errorf("failed to get stake changes: %w", err)
	}
	p.stakeChanges = append(p.stakeChanges, processedStakeChanges...)
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
