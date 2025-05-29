package flusher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	vmtypes "github.com/initia-labs/movevm/types"

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
}

func newValidatorEventProcessor() *validatorEventProcessor {
	return &validatorEventProcessor{
		stakeChanges: make(map[string]int64),
		validators:   make(map[string]bool),
	}
}

func (f *Flusher) processValidatorEvents(blockResults *mq.BlockResultMsg) error {
	for _, tx := range blockResults.Txs {
		if tx.ExecTxResults.Log == "tx parse error" {
			continue
		}

		processor := newValidatorEventProcessor()
		if err := processor.processTransactionEvents(&tx); err != nil {
			return fmt.Errorf("failed to process transaction events: %w", err)
		}

		f.blockStateUpdates.validators = processor.validators
		f.dbBatchInsert.AddValidatorBondedTokenTxs(processor.getStakeChanges(tx.Hash, blockResults.Height)...)
	}

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

func (p *validatorEventProcessor) handleEvent(event abci.Event) error {
	switch event.Type {
	case sdk.EventTypeMessage:
		p.handleMessageEvent(event)
	case mstakingtypes.EventTypeCreateValidator, mstakingtypes.EventTypeEditValidator:
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

func (p *validatorEventProcessor) handleMessageEvent(event abci.Event) {
	for _, attr := range event.Attributes {
		if attr.Key == sdk.AttributeKeyAction && attr.Value == "/cosmos.slashing.v1beta1.MsgUnjail" {
			p.validators[attr.Value] = true
		}
	}
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

func (p *validatorEventProcessor) handleRedelegateEvent(event abci.Event) {
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
			logger.Error().Msgf("invalid stake change key format: must be 'validatorAddr.denom'")
			continue
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
			ValidatorAddr: validatorAddr,
			TxId:          db.GetTxID(txHash, blockHeight),
			BlockHeight:   blockHeight,
			Tokens:        json.RawMessage(tokensJSON),
		})
	}

	return changes
}

func (f *Flusher) syncValidatorData(ctx context.Context) error {
	validatorAddresses := make([]string, 0, len(f.blockStateUpdates.validators))
	for addr := range f.blockStateUpdates.validators {
		validatorAddresses = append(validatorAddresses, addr)
	}

	return f.syncValidators(ctx, validatorAddresses)
}

func (f *Flusher) syncValidators(ctx context.Context, validatorAddresses []string) error {
	for _, validatorAddr := range validatorAddresses {
		valAcc, err := sdk.ValAddressFromBech32(validatorAddr)
		if err != nil {
			return fmt.Errorf("failed to convert validator address: %w", err)
		}

		accAddr := sdk.AccAddress(valAcc)
		vmAddr, _ := vmtypes.NewAccountAddressFromBytes(accAddr)

		f.dbBatchInsert.AddAccounts(db.Account{
			Address:   accAddr.String(),
			VMAddress: vmAddr.String(),
		})

		validator, err := f.rpcClient.Validator(ctx, validatorAddr)
		if err != nil {
			return fmt.Errorf("failed to fetch validator data: %w", err)
		}

		valInfo := validator.Validator
		if err := valInfo.UnpackInterfaces(f.encodingConfig.InterfaceRegistry); err != nil {
			return fmt.Errorf("failed to unpack validator info: %w", err)
		}

		consAddr, err := valInfo.GetConsAddr()
		if err != nil {
			return errors.Join(ErrorNonRetryable, err)
		}

		f.dbBatchInsert.AddValidators(
			db.NewValidator(
				valInfo,
				accAddr.String(),
				consAddr,
			),
		)
	}

	return nil
}
