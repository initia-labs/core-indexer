package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/getsentry/sentry-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func InsertProposalsIgnoreConflict(ctx context.Context, dbTx *gorm.DB, proposals []Proposal) error {
	span := sentry.StartSpan(ctx, "InsertProposalsIgnoreConflict")
	span.Description = "Insert proposals into the database"
	defer span.Finish()

	if len(proposals) == 0 {
		return nil
	}
	result := dbTx.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(&proposals, BatchSize)
	return result.Error
}

func UpdateProposalStatus(ctx context.Context, dbTx *gorm.DB, proposals []Proposal) error {
	span := sentry.StartSpan(ctx, "UpdateProposalStatus")
	span.Description = "Bulk update proposals into the database"
	defer span.Finish()

	if len(proposals) == 0 {
		return nil
	}
	for _, proposal := range proposals {
		result := dbTx.WithContext(ctx).
			Model(&Proposal{}).
			Where("id = ?", proposal.ID).
			Updates(map[string]any{
				"status":                proposal.Status,
				"voting_time":           proposal.VotingTime,
				"voting_end_time":       proposal.VotingEndTime,
				"resolved_height":       proposal.ResolvedHeight,
				"abstain":               proposal.Abstain,
				"yes":                   proposal.Yes,
				"no":                    proposal.No,
				"no_with_veto":          proposal.NoWithVeto,
				"resolved_voting_power": proposal.ResolvedVotingPower,
				"is_expedited":          proposal.IsExpedited,
			})
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func UpdatePrunedProposalStatus(ctx context.Context, dbTx *gorm.DB, proposals []Proposal) error {
	span := sentry.StartSpan(ctx, "UpdatePrunedProposalStatus")
	span.Description = "Bulk update pruned proposals into the database"
	defer span.Finish()

	if len(proposals) == 0 {
		return nil
	}
	for _, proposal := range proposals {
		result := dbTx.WithContext(ctx).
			Model(&Proposal{}).
			Where("id = ?", proposal.ID).
			Updates(map[string]any{
				"status": proposal.Status,
			})
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func UpdateOnlyExpeditedProposalStatus(ctx context.Context, dbTx *gorm.DB, proposals []Proposal) error {
	span := sentry.StartSpan(ctx, "UpdateOnlyExpeditedProposalStatus")
	span.Description = "Bulk update only expedited proposals into the database"
	defer span.Finish()

	if len(proposals) == 0 {
		return nil
	}
	for _, proposal := range proposals {
		result := dbTx.WithContext(ctx).
			Model(&Proposal{}).
			Where("id = ?", proposal.ID).
			Updates(map[string]any{
				"is_expedited": proposal.IsExpedited,
			})
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func UpdateProposalEmergencyNextTally(ctx context.Context, dbTx *gorm.DB, proposals map[int32]*time.Time) error {
	if len(proposals) == 0 {
		return nil
	}
	for proposal, nextTallyTime := range proposals {
		result := dbTx.WithContext(ctx).
			Model(&Proposal{}).
			Where("id = ?", proposal).
			Updates(map[string]any{
				"IsEmergency":            true,
				"EmergencyNextTallyTime": nextTallyTime,
			})
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func InsertProposalDeposits(ctx context.Context, dbTx *gorm.DB, proposalDeposits []ProposalDeposit) error {
	span := sentry.StartSpan(ctx, "InsertProposalDeposits")
	span.Description = "Bulk insert proposal_deposits into the database"
	defer span.Finish()

	return dbTx.WithContext(ctx).CreateInBatches(proposalDeposits, BatchSize).Error
}

func UpdateProposalTotalDeposit(ctx context.Context, dbTx *gorm.DB, totalDepositChanges map[int32][]sdk.Coin) error {
	for proposalID, depositChanges := range totalDepositChanges {
		var proposal Proposal
		result := dbTx.WithContext(ctx).
			Where("id = ?", proposalID).
			First(&proposal)
		if result.Error != nil {
			return result.Error
		}
		var totalDeposit sdk.Coins
		if err := json.Unmarshal(proposal.TotalDeposit, &totalDeposit); err != nil {
			return fmt.Errorf("failed to unmarshal total deposit of proposal %d - %w", proposalID, err)
		}
		totalDeposit = totalDeposit.Add(depositChanges...)
		totalDepositJSON, err := json.Marshal(totalDeposit)
		if err != nil {
			return fmt.Errorf("failed to marshal total deposit of proposal %d - %w", proposalID, err)
		}
		result = dbTx.WithContext(ctx).
			Model(&Proposal{}).
			Where("id = ?", proposalID).
			Updates(map[string]any{
				"total_deposit": totalDepositJSON,
			})
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func UpsertProposalVotes(ctx context.Context, dbTx *gorm.DB, proposalVotes []ProposalVote) error {
	if len(proposalVotes) == 0 {
		return nil
	}
	for _, vote := range proposalVotes {
		var existingVote ProposalVote
		result := dbTx.WithContext(ctx).
			Where("proposal_id = ? AND voter = ?", vote.ProposalID, vote.Voter).
			First(&existingVote)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				if err := dbTx.WithContext(ctx).Create(&vote).Error; err != nil {
					return fmt.Errorf("failed to insert proposal vote: %w", err)
				}
			} else {
				return fmt.Errorf("failed to check existing vote: %w", result.Error)
			}
		} else {
			if err := dbTx.WithContext(ctx).
				Model(&ProposalVote{}).
				Where("proposal_id = ? AND voter = ?", vote.ProposalID, vote.Voter).
				Updates(map[string]interface{}{
					"is_vote_weighted":  vote.IsVoteWeighted,
					"is_validator":      vote.IsValidator,
					"validator_address": vote.ValidatorAddress,
					"yes":               vote.Yes,
					"no":                vote.No,
					"abstain":           vote.Abstain,
					"no_with_veto":      vote.NoWithVeto,
					"transaction_id":    vote.TransactionID,
				}).Error; err != nil {
				return fmt.Errorf("failed to update proposal vote: %w", err)
			}
		}
	}
	return nil
}
