package services

import (
	"context"
	"strings"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

type ProposalService interface {
	GetProposals(ctx context.Context, pagination dto.PaginationQuery, proposer, statuses, types, search string) (*dto.ProposalsResponse, error)
	GetProposalsTypes(ctx context.Context) (*dto.ProposalsTypesResponse, error)
	GetProposalInfo(ctx context.Context, proposalId int) (*dto.ProposalInfoResponse, error)
	GetProposalVotes(ctx context.Context, pagination dto.PaginationQuery, proposalId int, search, answer string) (*dto.ProposalVotesResponse, error)
	GetProposalValidatorVotes(ctx context.Context, pagination dto.PaginationQuery, proposalId int, search, answer string) (*dto.ProposalValidatorVotesResponse, error)
	GetProposalAnswerCounts(ctx context.Context, proposalId int) (*dto.ProposalAnswerCountsResponse, error)
}

type proposalService struct {
	repo repositories.ProposalRepositoryI
}

func NewProposalService(repo repositories.ProposalRepositoryI) ProposalService {
	return &proposalService{
		repo: repo,
	}
}

func (s *proposalService) GetProposals(ctx context.Context, pagination dto.PaginationQuery, proposer, statuses, types, search string) (*dto.ProposalsResponse, error) {
	allowedStatuses := map[string]struct{}{
		"DepositPeriod": {},
		"VotingPeriod":  {},
		"Passed":        {},
		"Rejected":      {},
		"Failed":        {},
		"Inactive":      {},
		"Cancelled":     {},
	}

	statusesSlice := []string{}
	seenStatuses := map[string]struct{}{}

	if statuses != "" {
		for _, status := range strings.Split(statuses, ",") {
			v := strings.TrimSpace(status)
			if v == "" {
				continue
			}

			if _, ok := allowedStatuses[v]; !ok {
				return nil, apperror.NewBadRequest()
			}

			if _, exists := seenStatuses[v]; exists {
				return nil, apperror.NewDuplicateStatus()
			}

			seenStatuses[v] = struct{}{}
			statusesSlice = append(statusesSlice, v)
		}
	}

	var typesSlice []string
	if types != "" {
		for _, ty := range strings.Split(types, ",") {
			if v := strings.TrimSpace(ty); v != "" {
				typesSlice = append(typesSlice, v)
			}
		}
	}

	proposals, total, err := s.repo.SearchProposals(
		ctx,
		pagination,
		proposer,
		search,
		statusesSlice,
		typesSlice,
	)

	if err != nil {
		return nil, err
	}

	return &dto.ProposalsResponse{
		Proposals:  proposals,
		Pagination: dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}, nil
}

func (s *proposalService) GetProposalsTypes(ctx context.Context) (*dto.ProposalsTypesResponse, error) {
	return s.repo.GetAllProposalTypes(ctx)
}

func (s *proposalService) GetProposalInfo(ctx context.Context, proposalId int) (*dto.ProposalInfoResponse, error) {
	proposal, err := s.repo.GetProposalInfo(ctx, proposalId)
	if err != nil {
		return nil, err
	}

	return &dto.ProposalInfoResponse{
		Info: *proposal,
	}, nil
}

func (s *proposalService) GetProposalVotes(ctx context.Context, pagination dto.PaginationQuery, proposalId int, search, answer string) (*dto.ProposalVotesResponse, error) {
	votes, total, err := s.repo.GetProposalVotes(
		ctx,
		proposalId,
		int64(pagination.Limit),
		int64(pagination.Offset),
		search,
		answer,
	)
	if err != nil {
		return nil, err
	}

	return &dto.ProposalVotesResponse{
		Votes:      votes,
		Pagination: dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}, nil
}

func (s *proposalService) GetProposalValidatorVotes(ctx context.Context, pagination dto.PaginationQuery, proposalId int, search, answer string) (*dto.ProposalValidatorVotesResponse, error) {
	validatorVotes, err := s.repo.GetProposalValidatorVotes(ctx, proposalId)
	if err != nil {
		return nil, err
	}

	filteredVotes := make([]dto.ProposalVote, 0)
	for _, vote := range validatorVotes {
		if search != "" {
			moniker := strings.ToLower(vote.Validator.Moniker)
			validatorAddress := strings.ToLower(vote.Validator.ValidatorAddress)
			searchLower := strings.ToLower(search)

			if !strings.Contains(moniker, searchLower) && !strings.Contains(validatorAddress, searchLower) {
				continue
			}
		}

		if answer != "" && !matchValidatorVote(vote, answer) {
			continue
		}

		filteredVotes = append(filteredVotes, vote)
	}

	total := len(filteredVotes)
	start := int(pagination.Offset)
	end := int(pagination.Offset + pagination.Limit)

	if start >= total {
		start = total
	}
	if end > total {
		end = total
	}

	return &dto.ProposalValidatorVotesResponse{
		Votes:      filteredVotes[start:end],
		Pagination: dto.NewPaginationResponse(pagination.Offset, pagination.Limit, int64(total)),
	}, nil
}

func matchValidatorVote(vote dto.ProposalVote, answer string) bool {
	switch answer {
	case "yes":
		return vote.Yes == 1
	case "no":
		return vote.No == 1
	case "no_with_veto":
		return vote.NoWithVeto == 1
	case "abstain":
		return vote.Abstain == 1
	case "weighted":
		return vote.IsVoteWeighted
	case "did_not_vote":
		return vote.Yes == 0 && vote.No == 0 && vote.NoWithVeto == 0 && vote.Abstain == 0 && !vote.IsVoteWeighted
	default:
		return true
	}
}

func (s *proposalService) GetProposalAnswerCounts(ctx context.Context, proposalId int) (*dto.ProposalAnswerCountsResponse, error) {
	return s.repo.GetProposalAnswerCounts(ctx, proposalId)
}
