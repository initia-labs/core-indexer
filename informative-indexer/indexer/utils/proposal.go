package utils

import (
	"fmt"

	cosmosgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	cosmosgovv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/initia-labs/core-indexer/pkg/db"
)

func ParseProposalStatus(status cosmosgovv1.ProposalStatus) db.ProposalStatus {
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

func ParseProposalEndBlockAttributeValue(value string) (db.ProposalStatus, error) {
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

func IsExpeditedRejected(value string) bool {
	return value == cosmosgovtypes.AttributeValueExpeditedProposalRejected
}

func IsProposalResolved(status db.ProposalStatus) bool {
	return status == db.ProposalStatusInactive || status == db.ProposalStatusPassed || status == db.ProposalStatusRejected || status == db.ProposalStatusFailed || status == db.ProposalStatusCancelled
}

func IsProposalPruned(status db.ProposalStatus) bool {
	return status == db.ProposalStatusInactive || status == db.ProposalStatusCancelled
}
