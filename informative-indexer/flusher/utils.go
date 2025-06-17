package flusher

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	cosmosgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	cosmosgovv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/parser"
)

// findAttribute finds the first attribute with the given key and returns its value
func findAttribute(attributes []abci.EventAttribute, key string) (string, bool) {
	for _, attr := range attributes {
		if attr.Key == key {
			return attr.Value, true
		}
	}
	return "", false
}

// handleEventWithKey is a generic handler for events that need to decode data from a specific attribute
func handleEventWithKey[T any](event abci.Event, key string, flag *bool, store func(T)) error {
	if value, found := findAttribute(event.Attributes, key); found {
		e, err := parser.DecodeEvent[T](value)
		if err != nil {
			return fmt.Errorf("failed to decode event data: %w", err)
		}
		if flag != nil {
			*flag = true
		}
		store(e)
	}
	return nil
}

func parseProposalStatus(status cosmosgovv1.ProposalStatus) db.ProposalStatus {
	switch status {
	case cosmosgovv1.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD:
		return db.ProposalStatusDepositPeriod
	case cosmosgovv1.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD:
		return db.ProposalStatusVotingPeriod
	case cosmosgovv1.ProposalStatus_PROPOSAL_STATUS_PASSED:
		return db.ProposalStatusPassed
	case cosmosgovv1.ProposalStatus_PROPOSAL_STATUS_REJECTED:
		return db.ProposalStatusRejected
	case cosmosgovv1.ProposalStatus_PROPOSAL_STATUS_FAILED:
		return db.ProposalStatusFailed
	default:
		return db.ProposalStatusNil
	}
}

func parseProposalEndBlockAttributeValue(value string) (db.ProposalStatus, error) {
	switch value {
	case cosmosgovtypes.AttributeValueProposalPassed:
		return db.ProposalStatusPassed, nil
	case cosmosgovtypes.AttributeValueProposalRejected:
		return db.ProposalStatusRejected, nil
	case cosmosgovtypes.AttributeValueProposalFailed:
		return db.ProposalStatusFailed, nil
	case cosmosgovtypes.AttributeValueProposalDropped:
		return db.ProposalStatusInactive, nil
	default:
		return "", fmt.Errorf("unknown inactive proposal attribute: %s", value)
	}
}

func isProposalResolved(status db.ProposalStatus) bool {
	return status == db.ProposalStatusPassed || status == db.ProposalStatusRejected || status == db.ProposalStatusFailed || status == db.ProposalStatusCancelled
}
