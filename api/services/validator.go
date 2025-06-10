package services

import (
	"strconv"
	"strings"
	"sync"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/pkg/db"
)

type ValidatorService interface {
	GetValidators(pagination dto.PaginationQuery, isActive bool, sortBy, search string) (*dto.ValidatorsResponse, error)
	GetValidatorInfo(operatorAddr string) (*dto.ValidatorInfoResponse, error)
	GetValidatorUptime(operatorAddr string, blocks int) (*dto.ValidatorUptimeResponse, error)
}

type validatorService struct {
	repo      *repositories.ValidatorRepository
	blockRepo *repositories.BlockRepository
}

func NewValidatorService(repo *repositories.ValidatorRepository, blockRepo *repositories.BlockRepository) ValidatorService {
	return &validatorService{
		repo:      repo,
		blockRepo: blockRepo,
	}
}

func (s *validatorService) GetValidators(pagination dto.PaginationQuery, isActive bool, sortBy, search string) (*dto.ValidatorsResponse, error) {
	validators, total, err := s.repo.GetValidators(pagination, isActive, sortBy, search)
	if err != nil {
		return nil, err
	}

	validatorsByPower, err := s.repo.GetValidatorsByPower(&pagination, false)
	if err != nil {
		return nil, err
	}

	minCommissionRate := 1.0
	var active, inactive []db.Validator

	for _, val := range validatorsByPower {
		if val.IsActive == isActive {
			valRate, err := strconv.ParseFloat(val.CommissionRate, 64)
			if err == nil && valRate < minCommissionRate {
				minCommissionRate = valRate
			}
		}

		if val.IsActive {
			active = append(active, val)
		} else {
			inactive = append(inactive, val)
		}
	}

	totalVotingPower, rankMap, percent33Rank, percent66Rank := getTotalVotingPowerAndRank(active)

	validatorInfoItems := make([]dto.ValidatorInfo, 0, len(validators))
	for _, val := range validators {
		validatorInfo := flattenValidatorInfo(&val, rankMap)
		if isActive {
			validatorInfo.Uptime = val.Last100
		} else {
			validatorInfo.Uptime = 0
		}
		validatorInfoItems = append(validatorInfoItems, *validatorInfo)
	}

	return &dto.ValidatorsResponse{
		Items: validatorInfoItems,
		Metadata: dto.ValidatorsMetadata{
			ActiveCount:       len(active),
			InactiveCount:     len(inactive),
			MinCommissionRate: strconv.FormatFloat(minCommissionRate, 'f', -1, 64),
			Percent33Rank:     percent33Rank,
			Percent66Rank:     percent66Rank,
			TotalVotingPower:  strconv.FormatInt(totalVotingPower, 10),
		},
		Total: total,
	}, nil
}

func flattenValidatorInfo(validator *dto.ValidatorWithVoteCountModel, rankMap map[string]int) *dto.ValidatorInfo {
	validatorInfo := &dto.ValidatorInfo{
		AccountAddress:   validator.AccountID,
		CommissionRate:   validator.CommissionRate,
		ConsensusAddress: validator.ConsensusAddress,
		Details:          validator.Details,
		Identity:         validator.Identity,
		IsActive:         validator.IsActive,
		IsJailed:         validator.Jailed,
		Moniker:          strings.TrimSpace(validator.Moniker),
		ValidatorAddress: validator.OperatorAddress,
		VotingPower:      strconv.FormatInt(validator.VotingPower, 10),
		Website:          validator.Website,
	}

	if rank, exists := rankMap[validator.OperatorAddress]; exists {
		validatorInfo.Rank = rank
	} else {
		validatorInfo.Rank = 0
	}

	return validatorInfo
}

func getTotalVotingPowerAndRank(validators []db.Validator) (int64, map[string]int, int, int) {
	totalVotingPower := int64(0)
	cumulativeVotingPower := int64(0)
	rankMap := make(map[string]int)
	var percent33Rank, percent66Rank int

	for idx, val := range validators {
		totalVotingPower += val.VotingPower
		rankMap[val.OperatorAddress] = idx + 1
	}

	for idx, val := range validators {
		cumulativeVotingPower += val.VotingPower
		cumulativePercent := float64(cumulativeVotingPower) / float64(totalVotingPower)

		if cumulativePercent >= 0.33 && percent33Rank == 0 {
			percent33Rank = idx + 1
		}

		if cumulativePercent > 0.66 {
			percent66Rank = idx + 1
			break
		}
	}

	return totalVotingPower, rankMap, percent33Rank, percent66Rank
}

func (s *validatorService) GetValidatorInfo(operatorAddr string) (*dto.ValidatorInfoResponse, error) {
	validator, err := s.repo.GetValidatorRow(operatorAddr)
	if err != nil {
		return nil, err
	}

	activeValidators, err := s.repo.GetValidatorsByPower(nil, true)
	totalVotingPower, rankMap, _, _ := getTotalVotingPowerAndRank(activeValidators)
	validatorInfo := flattenValidatorInfo(&dto.ValidatorWithVoteCountModel{Validator: *validator}, rankMap)

	return &dto.ValidatorInfoResponse{
		Info:             *validatorInfo,
		TotalVotingPower: strconv.FormatInt(totalVotingPower, 10),
	}, nil
}

func (s *validatorService) GetValidatorUptime(operatorAddr string, blocks int) (*dto.ValidatorUptimeResponse, error) {
	latestBlock, err := s.blockRepo.GetLatestBlock()
	if err != nil {
		return nil, err
	}

	latestHeight := int64(latestBlock.Height)
	latestTimestamp := latestBlock.Timestamp

	total := int64(blocks)
	if total > latestHeight {
		total = latestHeight
	}

	minHeight := latestHeight - total + 1
	if minHeight < 1 {
		minHeight = 1
	}
	eventTimestampMin := latestTimestamp.AddDate(0, -3, 0)

	var proposedBlocks, validatorSignatures []dto.ValidatorBlockVote
	var slashEvents []dto.ValidatorUptimeEvent
	var validatorInfo *dto.ValidatorWithVoteCountModel
	var proposedBlocksErr, signaturesErr, eventsErr, validatorInfoErr error

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		proposedBlocks, proposedBlocksErr = s.repo.GetValidatorProposedBlocks(minHeight, latestHeight)
	}()

	go func() {
		defer wg.Done()
		validatorSignatures, signaturesErr = s.repo.GetValidatorCommitSignatures(operatorAddr, minHeight, latestHeight)
	}()

	go func() {
		defer wg.Done()
		slashEvents, eventsErr = s.repo.GetValidatorSlashEvents(operatorAddr, eventTimestampMin)
	}()

	go func() {
		defer wg.Done()
		validatorInfo, validatorInfoErr = s.repo.GetValidatorUptimeInfo(operatorAddr)
	}()

	wg.Wait()

	if proposedBlocksErr != nil {
		return nil, proposedBlocksErr
	}
	if signaturesErr != nil {
		return nil, signaturesErr
	}
	if eventsErr != nil {
		return nil, eventsErr
	}
	if validatorInfoErr != nil {
		return nil, validatorInfoErr
	}

	proposedBlocksMapping := make(map[int64]string)
	for _, block := range proposedBlocks {
		proposedBlocksMapping[block.Height] = block.Vote
	}

	commitSignaturesMapping := make(map[int64]string)
	for _, sig := range validatorSignatures {
		commitSignaturesMapping[sig.Height] = sig.Vote
	}

	validatorUptime := 0
	if validatorInfo != nil && validatorInfo.IsActive {
		validatorUptime = int(validatorInfo.Last100)
	}

	recent100Blocks := make([]dto.ValidatorBlockVote, 0, total)
	for i := latestHeight; i > latestHeight-total; i-- {
		vote := "VOTE"
		if val, exists := commitSignaturesMapping[i]; exists {
			vote = val
		} else if _, exists := proposedBlocksMapping[i+1]; exists && !containsKey(commitSignaturesMapping, i) || validatorUptime == 0 {
			vote = "ABSTAIN"
		}
		recent100Blocks = append(recent100Blocks, dto.ValidatorBlockVote{Height: i, Vote: vote})
	}

	uptime := dto.ValidatorUptimeSummary{
		SignedBlocks:   0,
		ProposedBlocks: 0,
		MissedBlocks:   0,
		Total:          int(total),
	}

	for _, block := range recent100Blocks {
		switch block.Vote {
		case "VOTE":
			uptime.SignedBlocks++
		case "PROPOSE":
			uptime.ProposedBlocks++
		default:
			uptime.MissedBlocks++
		}
	}

	return &dto.ValidatorUptimeResponse{
		Events:          slashEvents,
		Recent100Blocks: recent100Blocks[:100],
		Uptime:          uptime,
	}, nil
}

func containsKey(m map[int64]string, key int64) bool {
	_, exists := m[key]
	return exists
}
