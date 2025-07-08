package flusher

import (
	"encoding/json"
	"fmt"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/initia-labs/initia/app/params"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

type ValidatorTokenChange struct {
	ValidatorAddr string
	Denom         string
	Amount        int64
	TxHash        string
}

type validatorEventProcessor struct {
	stakeChanges map[string]int64
	validators   map[string]bool
	slashEvents  []db.ValidatorSlashEvent
}

func newValidatorEventProcessor() *validatorEventProcessor {
	return &validatorEventProcessor{
		stakeChanges: make(map[string]int64),
		validators:   make(map[string]bool),
	}
}

func (f *Flusher) processValidatorEvents(txResult *mq.TxResult, height int64, _ *db.Transaction) error {
	processor := newValidatorEventProcessor()
	if err := processor.processTransactionEvents(txResult); err != nil {
		return fmt.Errorf("failed to process transaction events: %w", err)
	}

	if err := processor.processSDKMessages(txResult, f.encodingConfig, height); err != nil {
		return fmt.Errorf("failed to process SDK messages: %w", err)
	}

	for addr := range processor.validators {
		f.stateUpdateManager.validators[addr] = true
	}
	f.dbBatchInsert.AddValidatorBondedTokenTxs(processor.getStakeChanges(txResult.Hash, height)...)
	f.dbBatchInsert.validatorSlashEvents = append(f.dbBatchInsert.validatorSlashEvents, processor.slashEvents...)

	return nil
}

func (p *validatorEventProcessor) processTransactionEvents(tx *mq.TxResult) error {
	for _, event := range tx.ExecTxResults.Events {
		if err := p.handleEvent(event); err != nil {
			return fmt.Errorf("failed to handle event %s: %w", event.Type, err)
		}
	}
	return nil
}

// processSDKMessages processes SDK transaction messages to identify entry points
func (p *validatorEventProcessor) processSDKMessages(tx *mq.TxResult, encodingConfig *params.EncodingConfig, height int64) error {
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
				Type:             fmt.Sprintf("%s", db.Unjailed),
			})
		}

	}

	return nil
}

func (p *validatorEventProcessor) handleEvent(event abci.Event) error {
	switch event.Type {
	case mstakingtypes.EventTypeCreateValidator:
		p.handleValidatorEvent(event)
	case mstakingtypes.EventTypeDelegate:
		p.handleDelegateEvent(event)
	case mstakingtypes.EventTypeUnbond:
		p.handleUnbondEvent(event)
	case mstakingtypes.EventTypeRedelegate:
		p.handleRedelegateEvent(event)
	}
	return nil
}

func (p *validatorEventProcessor) handleValidatorEvent(event abci.Event) {
	for _, attr := range event.Attributes {
		if attr.Key == mstakingtypes.AttributeKeyValidator {
			p.validators[attr.Value] = true
		}
	}
}

func (p *validatorEventProcessor) handleDelegateEvent(event abci.Event) {
	valAddr, coin := p.extractValidatorAndAmount(event)
	p.validators[valAddr] = true
	if valAddr != "" && coin != "" {
		if amount, denom, err := parser.ParseCoinAmount(coin); err == nil {
			p.updateStakeChange(valAddr, denom, amount)
		}
	}
}

func (p *validatorEventProcessor) handleUnbondEvent(event abci.Event) {
	valAddr, coin := p.extractValidatorAndAmount(event)
	p.validators[valAddr] = true
	if valAddr != "" && coin != "" {
		if amount, denom, err := parser.ParseCoinAmount(coin); err == nil {
			p.updateStakeChange(valAddr, denom, -amount)
		}
	}
}

func (p *validatorEventProcessor) handleRedelegateEvent(event abci.Event) error {
	srcValAddr, found := findAttribute(event.Attributes, mstakingtypes.AttributeKeySrcValidator)
	if !found {
		return fmt.Errorf("failed to find src validator address in %s", event.Type)
	}
	dstValAddr, found := findAttribute(event.Attributes, mstakingtypes.AttributeKeyDstValidator)
	if !found {
		return fmt.Errorf("failed to find dst validator address in %s", event.Type)
	}

	coin, found := findAttribute(event.Attributes, sdk.AttributeKeyAmount)
	if !found {
		return fmt.Errorf("failed to find amount in %s", event.Type)
	}

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
		if amount, denom, err := parser.ParseCoinAmount(coin); err == nil {
			p.updateStakeChange(srcValAddr, denom, -amount)
			p.updateStakeChange(dstValAddr, denom, amount)
		}
	}
}

func (p *validatorEventProcessor) extractValidatorAndAmount(event abci.Event) (valAddr, coin string) {
	for _, attr := range event.Attributes {
		switch attr.Key {
		case mstakingtypes.AttributeKeyValidator:
			valAddr = attr.Value
		case sdk.AttributeKeyAmount:
			coin = attr.Value
		}
	}
	return valAddr, coin
}

func (p *validatorEventProcessor) updateStakeChange(validatorAddr, denom string, amount int64) {
	key := fmt.Sprintf("%s.%s", validatorAddr, denom)
	p.stakeChanges[key] += amount
}

func (p *validatorEventProcessor) getStakeChanges(txHash string, blockHeight int64) []db.ValidatorBondedTokenChange {
	// Group changes by validator address
	validatorChanges := make(map[string][]map[string]string)

	for key, amount := range p.stakeChanges {
		parts := strings.Split(key, ".")
		if len(parts) != 2 {
			panic("invalid stake change key format: must be 'validatorAddr.denom'")
		}

		validatorAddr := parts[0]
		denom := parts[1]

		// Add token change to validator's list
		validatorChanges[validatorAddr] = append(validatorChanges[validatorAddr], map[string]string{
			"amount": fmt.Sprintf("%d", amount),
			"denom":  denom,
		})
	}

	// Convert grouped changes to ValidatorBondedTokenChange
	var changes []db.ValidatorBondedTokenChange
	for validatorAddr, tokens := range validatorChanges {
		tokensJSON, err := json.Marshal(tokens)
		if err != nil {
			panic(fmt.Sprintf("failed to marshal tokens: %v", err))
		}

		changes = append(changes, db.ValidatorBondedTokenChange{
			ValidatorAddress: validatorAddr,
			TransactionID:    db.GetTxID(txHash, blockHeight),
			BlockHeight:      blockHeight,
			Tokens:           tokensJSON,
		})
	}

	return changes
}
