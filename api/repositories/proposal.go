package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

var _ ProposalRepositoryI = &ProposalRepository{}

// ProposalRepository implements ProposalRepositoryI
type ProposalRepository struct {
	db                *gorm.DB
	countQueryTimeout time.Duration
}

func NewProposalRepository(db *gorm.DB, countQueryTimeout time.Duration) *ProposalRepository {
	return &ProposalRepository{
		db:                db,
		countQueryTimeout: countQueryTimeout,
	}
}

func (r *ProposalRepository) GetProposals(pagination *dto.PaginationQuery) ([]db.Proposal, error) {
	var proposals []db.Proposal

	desc := false
	if pagination != nil {
		desc = pagination.Reverse
	}

	if err := r.db.Model(&db.Proposal{}).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "id",
			},
			Desc: desc,
		}).
		Find(&proposals).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query all proposals")
		return nil, err
	}

	return proposals, nil
}

func (r *ProposalRepository) GetProposalVotesByValidator(operatorAddr string) ([]db.ProposalVote, error) {
	var votes []db.ProposalVote

	if err := r.db.Model(&db.ProposalVote{}).
		Preload("Proposal").
		Preload("Validator").
		Preload("Transaction").
		Preload("Transaction.Block").
		Where("validator_address = ?", operatorAddr).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "transaction_id",
			},
			Desc: true,
		}).Find(&votes).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query proposal votes for %s", operatorAddr)
		return nil, err
	}

	return votes, nil
}

func (r *ProposalRepository) SearchProposals(pagination dto.PaginationQuery, proposer, search string, statuses, types []string) ([]dto.ProposalSummary, int64, error) {
	var proposals []dto.ProposalSummary
	var total int64

	query := r.db.Model(&db.Proposal{}).
		Select("proposals.id, proposals.title, proposals.types, proposals.voting_end_time, proposals.deposit_end_time, proposals.resolved_height, proposals.status, proposals.is_expedited, proposals.is_emergency, proposals.proposer_id as proposer")

	if proposer != "" {
		query = query.Where("proposals.proposer_id = ?", proposer)
	}

	if len(statuses) > 0 {
		query = query.Where("proposals.status IN ?", statuses)
	}

	if search != "" {
		if id, err := strconv.Atoi(search); err == nil {
			query = query.Where("proposals.id = ?", id)
		} else {
			query = query.Where("proposals.title ILIKE ?", "%"+search+"%")
		}
	}

	if len(types) > 0 {
		query = query.Where("proposals.type IN ?", types)
	}

	if pagination.CountTotal {
		countQuery := query
		var err error
		total, err = db.CountWithTimeout(countQuery, r.countQueryTimeout)
		if err != nil {
			logger.Get().Error().Err(err).Msgf("Failed to query proposal count")
			return nil, 0, err
		}
	}

	if err := query.Order(clause.OrderByColumn{
		Column: clause.Column{
			Name: "proposals.id",
		},
		Desc: pagination.Reverse,
	}).
		Limit(int(pagination.Limit)).
		Offset(int(pagination.Offset)).
		Find(&proposals).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query proposals")
		return nil, 0, err
	}

	return proposals, total, nil
}

func (r *ProposalRepository) GetAllProposalTypes() (*dto.ProposalsTypesResponse, error) {
	var proposals []struct {
		Types json.RawMessage `gorm:"column:types"`
	}

	if err := r.db.Model(&db.Proposal{}).
		Select("types").
		Find(&proposals).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query all proposal types")
		return nil, err
	}

	typesSet := make(map[string]bool)
	for _, proposal := range proposals {
		var types []string
		if err := json.Unmarshal(proposal.Types, &types); err == nil {
			for _, t := range types {
				typesSet[t] = true
			}
		}

	}

	types := make(dto.ProposalsTypesResponse, 0, len(typesSet))
	for t := range typesSet {
		types = append(types, t)
	}

	sort.Slice(types, func(i, j int) bool {
		return compareProposalTypes(types[i], types[j]) < 0
	})

	return &types, nil
}

func compareProposalTypes(a, b string) int {
	if strings.HasPrefix(a, "/") && !strings.HasPrefix(b, "/") {
		return 1
	}
	if !strings.HasPrefix(a, "/") && strings.HasPrefix(b, "/") {
		return -1
	}
	return strings.Compare(a, b)
}

func (r *ProposalRepository) GetProposalInfo(id int) (*dto.ProposalInfo, error) {
	var proposal dto.ProposalInfoModel

	if err := r.db.Model(&db.Proposal{}).
		Select("proposals.*, proposals.proposer_id as proposer_address, proposals.resolved_voting_power as resolved_total_voting_power, c_tx.hash as created_tx_hash, c_block.height as created_height, c_block.timestamp as created_timestamp, r_block.height as resolved_height, r_block.timestamp as resolved_timestamp").
		Joins("LEFT JOIN transactions c_tx ON proposals.created_tx = c_tx.id").
		Joins("LEFT JOIN blocks c_block ON c_tx.block_height = c_block.height").
		Joins("LEFT JOIN blocks r_block ON proposals.resolved_height = r_block.height").
		Where("proposals.id = ?", id).
		First(&proposal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("proposal not found")
		}
		logger.Get().Error().Err(err).Msgf("Failed to query proposal info for %d", id)
		return nil, err
	}

	var types, messages, metadata json.RawMessage
	if err := json.Unmarshal([]byte(proposal.Types), &types); err != nil {
		types = nil
	}
	if err := json.Unmarshal([]byte(proposal.Messages), &messages); err != nil {
		messages = nil
	}
	if err := json.Unmarshal([]byte(proposal.Metadata), &metadata); err != nil {
		metadata = nil
	}

	var content dto.ProposalContent
	if proposal.Content != nil {
		if err := json.Unmarshal([]byte(*proposal.Content), &content); err != nil {
			content = dto.ProposalContent{}
		}
	} else {
		content = dto.ProposalContent{}
	}

	var totalDeposit dto.Coins
	if err := json.Unmarshal([]byte(proposal.TotalDeposit), &totalDeposit); err != nil {
		totalDeposit = nil
	}

	var deposits []dto.ProposalDepositModel
	if err := r.db.Model(&db.ProposalDeposit{}).
		Select("proposal_deposits.amount, proposal_deposits.depositor, transactions.hash as tx_hash, blocks.timestamp").
		Joins("LEFT JOIN transactions ON proposal_deposits.transaction_id = transactions.id").
		Joins("LEFT JOIN blocks ON transactions.block_height = blocks.height").
		Where("proposal_deposits.proposal_id = ?", id).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Name: "blocks.height",
			},
			Desc: true,
		}).Find(&deposits).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query proposal deposits for %d", id)
		return nil, err
	}

	proposalDeposits := make([]dto.ProposalDeposit, len(deposits))
	for i, d := range deposits {
		var amount dto.Coins
		if err := json.Unmarshal(d.Amount, &amount); err != nil {
			logger.Get().Error().Err(err).Msgf("Failed to query proposal deposits for %d", id)
		}
		proposalDeposits[i] = dto.ProposalDeposit{
			Amount:    amount,
			Depositor: d.Depositor,
			TxHash:    fmt.Sprintf("%x", d.TxHash),
			Timestamp: d.Timestamp,
		}
	}

	return &dto.ProposalInfo{
		Id:                       proposal.Id,
		Proposer:                 proposal.Proposer,
		Types:                    types,
		Title:                    proposal.Title,
		Description:              proposal.Description,
		Status:                   proposal.Status,
		FailedReason:             proposal.FailedReason,
		SubmitTime:               proposal.SubmitTime,
		DepositEndTime:           proposal.DepositEndTime,
		VotingTime:               proposal.VotingTime,
		VotingEndTime:            proposal.VotingEndTime,
		Content:                  content,
		Messages:                 messages,
		IsExpedited:              proposal.IsExpedited,
		IsEmergency:              proposal.IsEmergency,
		TotalDeposit:             totalDeposit,
		Version:                  proposal.Version,
		CreatedTxHash:            fmt.Sprintf("%x", proposal.CreatedTxHash),
		CreatedHeight:            proposal.CreatedHeight,
		CreatedTimestamp:         proposal.CreatedTimestamp,
		ResolvedHeight:           proposal.ResolvedHeight,
		ResolvedTimestamp:        proposal.ResolvedTimestamp,
		Metadata:                 proposal.Metadata,
		ProposalDeposits:         proposalDeposits,
		Yes:                      proposal.Yes,
		Abstain:                  proposal.Abstain,
		No:                       proposal.No,
		NoWithVeto:               proposal.NoWithVeto,
		ResolvedTotalVotingPower: proposal.ResolvedTotalVotingPower,
	}, nil
}

func (r *ProposalRepository) GetProposalVotes(proposalId int, limit, offset int64, search, answer string) ([]dto.ProposalVote, int64, error) {
	var votes []dto.ProposalVoteModel

	query := r.db.Model(&db.ProposalVote{}).
		Select("proposal_votes.proposal_id, proposal_votes.yes, proposal_votes.no, proposal_votes.no_with_veto, proposal_votes.abstain, proposal_votes.is_vote_weighted, proposal_votes.voter, transactions.hash as tx_hash, blocks.timestamp, validators.operator_address as validator_address, validators.moniker as validator_moniker, validators.identity as validator_identity").
		Joins("LEFT JOIN transactions ON proposal_votes.transaction_id = transactions.id").
		Joins("LEFT JOIN blocks ON transactions.block_height = blocks.height").
		Joins("LEFT JOIN validators ON proposal_votes.validator_address = validators.operator_address").
		Where("proposal_votes.proposal_id = ?", proposalId)

	if search != "" {
		query = query.Where("proposal_votes.voter = ?", search)
	}

	switch answer {
	case "yes":
		query = query.Where("proposal_votes.yes = ?", 1)
	case "no":
		query = query.Where("proposal_votes.no = ?", 1)
	case "no_with_veto":
		query = query.Where("proposal_votes.no_with_veto = ?", 1)
	case "abstain":
		query = query.Where("proposal_votes.abstain = ?", 1)
	case "weighted":
		query = query.Where("proposal_votes.is_vote_weighted = ?", true)
	}

	var total int64
	countQuery := query
	if err := countQuery.Count(&total).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query proposal votes count for %d", proposalId)
		return nil, 0, err
	}

	if err := query.Order(clause.OrderBy{
		Columns: []clause.OrderByColumn{
			{
				Column: clause.Column{
					Name: "transactions.block_height",
				},
				Desc: true,
			},
			{
				Column: clause.Column{
					Name: "transactions.block_index",
				},
				Desc: true,
			},
		},
	}).
		Limit(int(limit)).
		Offset(int(offset)).
		Find(&votes).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query proposal votes for %d", proposalId)
		return nil, 0, err
	}

	result := make([]dto.ProposalVote, len(votes))
	for idx, vote := range votes {
		var validator *dto.ProposalValidatorVote
		if vote.ValidatorAddr != nil && vote.ValidatorMoniker != nil && vote.ValidatorIdentity != nil {
			validator = &dto.ProposalValidatorVote{
				Moniker:          *vote.ValidatorMoniker,
				Identity:         *vote.ValidatorIdentity,
				ValidatorAddress: *vote.ValidatorAddr,
			}
		}

		txHash := fmt.Sprintf("%x", vote.TxHash)
		result[idx] = dto.ProposalVote{
			ProposalId:     vote.ProposalID,
			Yes:            vote.Yes,
			No:             vote.No,
			NoWithVeto:     vote.NoWithVeto,
			Abstain:        vote.Abstain,
			IsVoteWeighted: vote.IsVoteWeighted,
			Voter:          vote.Voter,
			TxHash:         &txHash,
			Timestamp:      &vote.Timestamp,
			Validator:      validator,
		}
	}

	return result, total, nil
}

func (r *ProposalRepository) GetProposalValidatorVotes(proposalId int) ([]dto.ProposalVote, error) {
	var validators []dto.ProposalVoteValidatorInfoModel

	if err := r.db.Model(&db.Validator{}).
		Select("operator_address, moniker, identity").
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "voting_power DESC NULLS LAST", Raw: true},
		}).
		Find(&validators).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query validators")
		return nil, err
	}

	var votes []dto.ProposalValidatorVoteModel

	if err := r.db.Model(&db.ProposalVote{}).
		Select("validators.operator_address as validator_address, "+
			"proposal_votes.yes, proposal_votes.no, "+
			"proposal_votes.no_with_veto, proposal_votes.abstain, "+
			"proposal_votes.is_vote_weighted, proposal_votes.voter, "+
			"transactions.hash as tx_hash, blocks.timestamp").
		Joins("JOIN validators ON proposal_votes.validator_address = validators.operator_address").
		Joins("JOIN transactions ON proposal_votes.transaction_id = transactions.id").
		Joins("JOIN blocks ON transactions.block_height = blocks.height").
		Where("proposal_votes.proposal_id = ? AND proposal_votes.is_validator = ?", proposalId, true).
		Order(clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{
					Column: clause.Column{
						Name: "transactions.block_height",
					},
					Desc: true,
				},
				{
					Column: clause.Column{
						Name: "transactions.block_index",
					},
					Desc: true,
				},
			},
		}).
		Find(&votes).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query validator votes for proposal id: %d", proposalId)
		return nil, err
	}

	votesByValidator := make(map[string]dto.ProposalVote)
	for _, vote := range votes {
		votesByValidator[vote.ValidatorAddress] = dto.ProposalVote{
			Yes:            vote.Yes,
			No:             vote.No,
			NoWithVeto:     vote.NoWithVeto,
			Abstain:        vote.Abstain,
			IsVoteWeighted: vote.IsVoteWeighted,
			Voter:          vote.Voter,
			TxHash:         vote.TxHash,
			Timestamp:      vote.Timestamp,
		}
	}

	result := make([]dto.ProposalVote, 0, len(validators))
	for _, validator := range validators {
		vote, hasVoted := votesByValidator[validator.OperatorAddress]

		validatorVote := dto.ProposalVote{
			ProposalId: proposalId,
			Validator: &dto.ProposalValidatorVote{
				Moniker:          validator.Moniker,
				Identity:         validator.Identity,
				ValidatorAddress: validator.OperatorAddress,
			},
		}

		if hasVoted {
			txHash := fmt.Sprintf("%x", *vote.TxHash)
			validatorVote.Yes = vote.Yes
			validatorVote.No = vote.No
			validatorVote.NoWithVeto = vote.NoWithVeto
			validatorVote.Abstain = vote.Abstain
			validatorVote.IsVoteWeighted = vote.IsVoteWeighted
			validatorVote.Voter = vote.Voter
			validatorVote.TxHash = &txHash
			validatorVote.Timestamp = vote.Timestamp
		} else {
			validatorVote.Yes = 0
			validatorVote.No = 0
			validatorVote.NoWithVeto = 0
			validatorVote.Abstain = 0
			validatorVote.IsVoteWeighted = false
			validatorVote.Voter = ""
			validatorVote.TxHash = nil
			validatorVote.Timestamp = nil
		}

		result = append(result, validatorVote)
	}

	return result, nil
}

func (r *ProposalRepository) GetProposalAnswerCounts(id int) (*dto.ProposalAnswerCountsResponse, error) {
	var totalValidators int64
	if err := r.db.Model(&db.Validator{}).Count(&totalValidators).Error; err != nil {
		logger.Get().Error().Err(err).Msg("Failed to query total validators")
		return nil, err
	}

	var votes []dto.ProposalAnswerCountsModel
	if err := r.db.Model(&db.ProposalVote{}).
		Select("yes, no, no_with_veto, abstain, is_vote_weighted, is_validator").
		Where("proposal_id = ?", id).
		Find(&votes).Error; err != nil {
		logger.Get().Error().Err(err).Msgf("Failed to query proposal votes for id: %d", id)
		return nil, err
	}

	response := &dto.ProposalAnswerCountsResponse{
		All: dto.ProposalAnswerCounts{},
		Validator: dto.ProposalValidatorAnswerCounts{
			TotalValidators: int(totalValidators),
		},
	}

	for _, vote := range votes {
		if vote.Yes == 1 {
			response.All.Yes++
			response.All.Total++
			if vote.IsValidator {
				response.Validator.Yes++
				response.Validator.Total++
			}
			continue
		}
		if vote.No == 1 {
			response.All.No++
			response.All.Total++
			if vote.IsValidator {
				response.Validator.No++
				response.Validator.Total++
			}
			continue
		}
		if vote.NoWithVeto == 1 {
			response.All.NoWithVeto++
			response.All.Total++
			if vote.IsValidator {
				response.Validator.NoWithVeto++
				response.Validator.Total++
			}
			continue
		}
		if vote.Abstain == 1 {
			response.All.Abstain++
			response.All.Total++
			if vote.IsValidator {
				response.Validator.Abstain++
				response.Validator.Total++
			}
			continue
		}
		if vote.IsVoteWeighted {
			response.All.Weighted++
			response.All.Total++
			if vote.IsValidator {
				response.Validator.Weighted++
				response.Validator.Total++
			}
			continue
		}
	}

	response.Validator.DidNotVote = response.Validator.TotalValidators - response.Validator.Total

	return response, nil
}
