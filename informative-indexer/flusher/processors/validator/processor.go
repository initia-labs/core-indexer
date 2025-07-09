package validator

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/initia-labs/initia/app/params"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	"github.com/initia-labs/core-indexer/informative-indexer/flusher/processors"
	statetracker "github.com/initia-labs/core-indexer/informative-indexer/flusher/state-tracker"
	"github.com/initia-labs/core-indexer/informative-indexer/flusher/types"
	"github.com/initia-labs/core-indexer/informative-indexer/flusher/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

var _ processors.Processor = &Processor{}

func (p *Processor) InitProcessor() {
	p.stakeChanges = make(map[string]int64)
	p.validators = make(map[string]bool)
	p.slashEvents = make([]db.ValidatorSlashEvent, 0)
}

func (p *Processor) Name() string {
	return "validator"
}

// processSDKMessages processes SDK transaction messages to identify entry points
func (p *Processor) ProcessSDKMessages(tx *mq.TxResult, height int64, encodingConfig *params.EncodingConfig) error {
	if !tx.ExecTxResults.IsOK() {
		return nil
	}

	sdkTx, err := encodingConfig.TxConfig.TxDecoder()(tx.Tx)
	if err != nil {
		return fmt.Errorf("failed to decode SDK transaction: %w", err)
	}

	for _, msg := range sdkTx.GetMsgs() {
		switch msg := msg.(type) {
		case *slashingtypes.MsgUnjail:
			p.validators[msg.ValidatorAddr] = true
			p.slashEvents = append(p.slashEvents, db.ValidatorSlashEvent{
				ValidatorAddress: msg.ValidatorAddr,
				BlockHeight:      height,
				Type:             string(db.Unjailed),
			})
		}
	}

	return nil
}

func (p *Processor) ProcessTransactionEvents(tx *mq.TxResult) error {
	for _, event := range tx.ExecTxResults.Events {
		if err := p.handleEvent(event); err != nil {
			return fmt.Errorf("failed to handle event %s: %w", event.Type, err)
		}
	}
	return nil
}

func (p *Processor) TrackState(txHash string, blockHeight int64, stateUpdateManager *statetracker.StateUpdateManager) error {
	for addr := range p.validators {
		stateUpdateManager.Validators[addr] = true
	}
	processedStakeChanges, err := processStakeChanges(&p.stakeChanges, txHash, blockHeight)
	if err != nil {
		return fmt.Errorf("failed to get stake changes: %w", err)
	}
	stateUpdateManager.DBBatchInsert.AddValidatorBondedTokenTxs(processedStakeChanges...)
	stateUpdateManager.DBBatchInsert.AddValidatorSlashEvents(p.slashEvents...)

	return nil
}

func (p *Processor) handleEvent(event abci.Event) error {
	switch event.Type {
	case sdk.EventTypeMessage:
		return p.handleMessageEvent(event)
	case mstakingtypes.EventTypeCreateValidator:
		return p.handleValidatorEvent(event)
	case mstakingtypes.EventTypeDelegate:
		return p.handleDelegateEvent(event)
	case mstakingtypes.EventTypeUnbond:
		return p.handleUnbondEvent(event)
	case mstakingtypes.EventTypeRedelegate:
		return p.handleRedelegateEvent(event)
	}
	return nil
}

func (p *Processor) handleMessageEvent(event abci.Event) error {
	if found := utils.FindAttributeWithValue(event.Attributes, sdk.AttributeKeyAction, types.AttributeValueActionUnjail); found {
		p.validators[types.AttributeValueActionUnjail] = true
	}
	return nil
}

func (p *Processor) handleValidatorEvent(event abci.Event) error {
	if value, found := utils.FindAttribute(event.Attributes, mstakingtypes.AttributeKeyValidator); found {
		p.validators[value] = true
	}
	return nil
}

func (p *Processor) handleDelegateEvent(event abci.Event) error {
	valAddr, coin, err := extractValidatorAndAmount(event)
	if err != nil {
		return fmt.Errorf("failed to extract validator and amount: %w", err)
	}
	p.validators[valAddr] = true
	if valAddr != "" && coin != "" {
		amount, denom, err := parser.ParseCoinAmount(coin)
		if err != nil {
			return fmt.Errorf("failed to parse coin amount: %w", err)
		}
		p.updateStakeChange(valAddr, denom, amount)
	}
	return nil
}

func (p *Processor) handleUnbondEvent(event abci.Event) error {
	valAddr, coin, err := extractValidatorAndAmount(event)
	if err != nil {
		return fmt.Errorf("failed to extract validator and amount: %w", err)
	}
	p.validators[valAddr] = true
	if valAddr != "" && coin != "" {
		amount, denom, err := parser.ParseCoinAmount(coin)
		if err != nil {
			return fmt.Errorf("failed to parse coin amount: %w", err)
		}
		p.updateStakeChange(valAddr, denom, -amount)
	}
	return nil
}

func (p *Processor) handleRedelegateEvent(event abci.Event) error {
	var srcValAddr, dstValAddr, coin string
	p.validators[srcValAddr] = true
	p.validators[dstValAddr] = true
	for _, attr := range event.Attributes {
		switch attr.Key {
		case mstakingtypes.AttributeKeySrcValidator:
			srcValAddr = attr.Value
		case mstakingtypes.AttributeKeyDstValidator:
			dstValAddr = attr.Value
		case sdk.AttributeKeyAmount:
			coin = attr.Value
		}
	}

	if srcValAddr != "" && dstValAddr != "" && coin != "" {
		amount, denom, err := parser.ParseCoinAmount(coin)
		if err != nil {
			return fmt.Errorf("failed to parse coin amount: %w", err)
		}
		p.updateStakeChange(srcValAddr, denom, -amount)
		p.updateStakeChange(dstValAddr, denom, amount)
	}
	return nil
}

func (p *Processor) updateStakeChange(validatorAddr, denom string, amount int64) {
	key := fmt.Sprintf("%s.%s", validatorAddr, denom)
	p.stakeChanges[key] += amount
}
