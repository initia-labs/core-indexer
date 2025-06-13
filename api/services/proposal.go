package services

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"strings"
)

type ProposalService interface {
	GetProposals(pagination dto.PaginationQuery, proposer, statuses, types, search string) (*dto.ProposalsResponse, error)
	GetProposalsTypes() (*dto.ProposalsTypesResponse, error)
	GetProposalInfo(proposalId int) (*dto.ProposalInfoResponse, error)
	GetProposalVotes(pagination dto.PaginationQuery, proposalId int, search, answer string) (*dto.ProposalVotesResponse, error)
	GetProposalValidatorVotes(pagination dto.PaginationQuery, proposalId int, search, answer string) (*dto.ProposalValidatorVotesResponse, error)
	GetProposalAnswerCounts(proposalId int) (*dto.ProposalAnswerCountsResponse, error)
}

type proposalService struct {
	repo repositories.ProposalRepositoryI
}

func NewProposalService(repo repositories.ProposalRepositoryI) ProposalService {
	return &proposalService{
		repo: repo,
	}
}

func (s *proposalService) GetProposals(pagination dto.PaginationQuery, proposer, statuses, types, search string) (*dto.ProposalsResponse, error) {
	var statusesSlice []string
	if statuses != "" {
		statusesSlice = strings.Split(statuses, ",")
	}

	var typesSlice []string
	if types != "" {
		typesSlice = strings.Split(types, ",")
	}

	proposals, total, err := s.repo.SearchProposals(
		pagination.Limit,
		pagination.Offset,
		proposer,
		search,
		statusesSlice,
		typesSlice,
	)

	if err != nil {
		return nil, err
	}

	return &dto.ProposalsResponse{
		Items: proposals,
		Total: total,
	}, nil
}

func (s *proposalService) GetProposalsTypes() (*dto.ProposalsTypesResponse, error) {
	return s.repo.GetAllProposalTypes()
}

func (s *proposalService) GetProposalInfo(proposalId int) (*dto.ProposalInfoResponse, error) {
	proposal, err := s.repo.GetProposalInfo(proposalId)
	if err != nil {
		return nil, err
	}

	return &dto.ProposalInfoResponse{
		Info: *proposal,
	}, nil
}

func (s *proposalService) GetProposalVotes(pagination dto.PaginationQuery, proposalId int, search, answer string) (*dto.ProposalVotesResponse, error) {
	votes, total, err := s.repo.GetProposalVotes(
		proposalId,
		pagination.Limit,
		pagination.Offset,
		search,
		answer,
	)
	if err != nil {
		return nil, err
	}

	return &dto.ProposalVotesResponse{
		Items: votes,
		Total: total,
	}, nil
}

func (s *proposalService) GetProposalValidatorVotes(pagination dto.PaginationQuery, proposalId int, search, answer string) (*dto.ProposalValidatorVotesResponse, error) {
	validatorVotes, err := s.repo.GetProposalValidatorVotes(proposalId)
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
		Items: filteredVotes[start:end],
		Total: int64(total),
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

func (s *proposalService) GetProposalAnswerCounts(proposalId int) (*dto.ProposalAnswerCountsResponse, error) {
	return s.repo.GetProposalAnswerCounts(proposalId)
}
