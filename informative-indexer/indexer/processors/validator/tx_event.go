package validator

import (
	"fmt"
	"strings"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/utils"
)

func (p *Processor) handleEvent(event abci.Event) error {
	switch event.Type {
	case mstakingtypes.EventTypeCreateValidator:
		return p.handleValidatorEvent(event)
	case mstakingtypes.EventTypeDelegate:
		return p.handleDelegateEvent(event)
	case mstakingtypes.EventTypeUnbond:
		return p.handleUnbondEvent(event)
	case mstakingtypes.EventTypeRedelegate:
		return p.handleRedelegateEvent(event)
	default:
		return nil
	}
}

func (p *Processor) handleValidatorEvent(event abci.Event) error {
	if value, found := utils.FindAttribute(event.Attributes, mstakingtypes.AttributeKeyValidator); found {
		p.validators[value] = true
	}
	return nil
}

func (p *Processor) handleDelegateEvent(event abci.Event) error {
	valAddr, amount, err := extractValidatorAndAmount(event)
	if err != nil {
		return fmt.Errorf("failed to extract validator and amount: %w", err)
	}
	p.validators[valAddr] = true
	if valAddr != "" && amount != "" {
		coins, err := sdk.ParseCoinsNormalized(amount)
		if err != nil {
			return fmt.Errorf("failed to parse amount: %w", err)
		}
		for _, coin := range coins {
			p.updateTxStakeChange(valAddr, coin.Denom, coin.Amount.Int64())
		}
	}
	return nil
}

func (p *Processor) handleUnbondEvent(event abci.Event) error {
	valAddr, amount, err := extractValidatorAndAmount(event)
	if err != nil {
		return fmt.Errorf("failed to extract validator and amount: %w", err)
	}
	p.validators[valAddr] = true
	if valAddr != "" && amount != "" {
		coins, err := sdk.ParseCoinsNormalized(amount)
		if err != nil {
			return fmt.Errorf("failed to parse amount: %w", err)
		}
		for _, coin := range coins {
			p.updateTxStakeChange(valAddr, coin.Denom, -coin.Amount.Int64())
		}
	}
	return nil
}

func (p *Processor) handleRedelegateEvent(event abci.Event) error {
	srcValAddr, found := utils.FindAttribute(event.Attributes, mstakingtypes.AttributeKeySrcValidator)
	if !found {
		return fmt.Errorf("failed to find src validator address in %s", event.Type)
	}
	dstValAddr, found := utils.FindAttribute(event.Attributes, mstakingtypes.AttributeKeyDstValidator)
	if !found {
		return fmt.Errorf("failed to find dst validator address in %s", event.Type)
	}

	amount, found := utils.FindAttribute(event.Attributes, sdk.AttributeKeyAmount)
	if !found {
		return fmt.Errorf("failed to find amount in %s", event.Type)
	}

	for _, attr := range event.Attributes {
		switch attr.Key {
		case mstakingtypes.AttributeKeySrcValidator:
			srcValAddr = strings.ToLower(attr.Value)
		case mstakingtypes.AttributeKeyDstValidator:
			dstValAddr = strings.ToLower(attr.Value)
		case sdk.AttributeKeyAmount:
			amount = attr.Value
		}
	}

	if srcValAddr != "" && dstValAddr != "" && amount != "" {
		coins, err := sdk.ParseCoinsNormalized(amount)
		if err != nil {
			return fmt.Errorf("failed to parse amount: %w", err)
		}
		for _, coin := range coins {
			p.updateTxStakeChange(srcValAddr, coin.Denom, -coin.Amount.Int64())
			p.updateTxStakeChange(dstValAddr, coin.Denom, coin.Amount.Int64())
		}
	}
	return nil
}

func (p *Processor) updateTxStakeChange(validatorAddr, denom string, amount int64) {
	key := fmt.Sprintf("%s.%s", validatorAddr, denom)
	p.txProcessor.txStakeChanges[key] += amount
}
