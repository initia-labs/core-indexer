package validator

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/utils"
	"github.com/initia-labs/core-indexer/pkg/parser"
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
		p.updateTxStakeChange(valAddr, denom, amount)
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
		p.updateTxStakeChange(valAddr, denom, -amount)
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

	coin, found := utils.FindAttribute(event.Attributes, sdk.AttributeKeyAmount)
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
		amount, denom, err := parser.ParseCoinAmount(coin)
		if err != nil {
			return fmt.Errorf("failed to parse coin amount: %w", err)
		}
		p.updateTxStakeChange(srcValAddr, denom, -amount)
		p.updateTxStakeChange(dstValAddr, denom, amount)
	}
	return nil
}

func (p *Processor) updateTxStakeChange(validatorAddr, denom string, amount int64) {
	key := fmt.Sprintf("%s.%s", validatorAddr, denom)
	p.txProcessor.txStakeChanges[key] += amount
}
