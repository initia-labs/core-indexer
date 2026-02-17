package services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/utils"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/logger"
)

type ValidatorService interface {
	GetValidators(pagination dto.PaginationQuery, status dto.ValidatorStatusFilter, sortBy, search string) (*dto.ValidatorsResponse, error)
	GetValidatorInfo(operatorAddr string) (*dto.ValidatorInfoResponse, error)
	GetValidatorUptime(operatorAddr string, blocks int) (*dto.ValidatorUptimeResponse, error)
	GetValidatorDelegationTxs(pagination dto.PaginationQuery, operatorAddr string) (*dto.ValidatorDelegationRelatedTxsResponse, error)
	GetValidatorProposedBlocks(pagination dto.PaginationQuery, operatorAddr string) (*dto.ValidatorProposedBlocksResponse, error)
	GetValidatorHistoricalPowers(operatorAddr string) (*dto.ValidatorHistoricalPowersResponse, error)
	GetValidatorVotedProposals(pagination dto.PaginationQuery, operatorAddr, search, answer string) (*dto.ValidatorVotedProposalsResponse, error)
	GetValidatorAnswerCounts(operatorAddr string) (*dto.ValidatorAnswerCountsResponse, error)
}

type validatorService struct {
	repo           repositories.ValidatorRepositoryI
	blockRepo      repositories.BlockRepositoryI
	proposalRepo   repositories.ProposalRepositoryI
	keybaseService *KeybaseService
}

func NewValidatorService(repo repositories.ValidatorRepositoryI, blockRepo repositories.BlockRepositoryI, proposalRepo repositories.ProposalRepositoryI, keybaseService *KeybaseService) ValidatorService {
	return &validatorService{
		repo:           repo,
		blockRepo:      blockRepo,
		proposalRepo:   proposalRepo,
		keybaseService: keybaseService,
	}
}

func (s *validatorService) GetValidators(pagination dto.PaginationQuery, status dto.ValidatorStatusFilter, sortBy, search string) (*dto.ValidatorsResponse, error) {
	validators, total, err := s.repo.GetValidators(pagination, status, sortBy, search)
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
		shouldInclude := false
		switch status {
		case dto.ValidatorStatusFilterAll:
			shouldInclude = true
		case dto.ValidatorStatusFilterActive:
			shouldInclude = val.IsActive
		case dto.ValidatorStatusFilterInactive:
			shouldInclude = !val.IsActive
		}

		if shouldInclude {
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

	// Get block statistics for all validators
	operatorAddresses := make([]string, 0, len(validators))
	for _, val := range validators {
		operatorAddresses = append(operatorAddresses, val.OperatorAddress)
	}

	blockStats, err := s.repo.GetValidatorBlockStats(operatorAddresses)
	if err != nil {
		return nil, err
	}

	validatorInfoItems := make([]dto.ValidatorInfo, 0, len(validators))
	for _, val := range validators {
		validatorInfo := flattenValidatorInfo(&val, rankMap)
		if status == dto.ValidatorStatusFilterActive || (status == dto.ValidatorStatusFilterAll && val.IsActive) {
			validatorInfo.Uptime = val.Last100
		} else {
			validatorInfo.Uptime = 0
		}

		// Add block statistics
		if stats, exists := blockStats[val.OperatorAddress]; exists {
			validatorInfo.TotalBlocks = stats.TotalBlocks
			validatorInfo.SignedBlocks = stats.SignedBlocks
		}

		// Add image URL based on identity (keybase format) - cached
		if val.Identity != "" {
			validatorInfo.Image = s.keybaseService.GetImageURL(val.Identity)
		}

		validatorInfoItems = append(validatorInfoItems, *validatorInfo)
	}

	return &dto.ValidatorsResponse{
		ValidatorsInfo: validatorInfoItems,
		Metadata: dto.ValidatorsMetadata{
			ActiveCount:       len(active),
			InactiveCount:     len(inactive),
			MinCommissionRate: strconv.FormatFloat(minCommissionRate, 'f', -1, 64),
			Percent33Rank:     percent33Rank,
			Percent66Rank:     percent66Rank,
			TotalVotingPower:  strconv.FormatInt(totalVotingPower, 10),
		},
		Pagination: dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
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
	if err != nil {
		return nil, err
	}

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

	var proposedBlocks, validatorSignatures []dto.ValidatorBlockVoteModel
	var slashEvents []dto.ValidatorUptimeEventModel
	var validatorInfo *dto.ValidatorWithVoteCountModel
	var proposedBlocksErr, signaturesErr, eventsErr, validatorInfoErr error

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		proposedBlocks, proposedBlocksErr = s.repo.GetValidatorBlockVoteByBlockLimit(minHeight, latestHeight)
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

	recent100Blocks := make([]dto.ValidatorBlockVoteModel, 0, total)
	for i := latestHeight; i > latestHeight-total; i-- {
		vote := "VOTE"
		if val, exists := commitSignaturesMapping[i]; exists {
			vote = val
		} else if _, exists := proposedBlocksMapping[i+1]; exists && !utils.ContainsKey(commitSignaturesMapping, i) || validatorUptime == 0 {
			vote = "ABSTAIN"
		}
		recent100Blocks = append(recent100Blocks, dto.ValidatorBlockVoteModel{Height: i, Vote: vote})
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

	maxBlocks := len(recent100Blocks)
	if maxBlocks > 100 {
		maxBlocks = 100
	}

	return &dto.ValidatorUptimeResponse{
		Events:          slashEvents,
		Recent100Blocks: recent100Blocks[:maxBlocks],
		Uptime:          uptime,
	}, nil
}

func (s *validatorService) GetValidatorDelegationTxs(pagination dto.PaginationQuery, operatorAddr string) (*dto.ValidatorDelegationRelatedTxsResponse, error) {
	tokenChanges, total, err := s.repo.GetValidatorBondedTokenChanges(pagination, operatorAddr)
	if err != nil {
		return nil, err
	}

	items := make([]dto.ValidatorDelegationRelatedTx, 0, len(tokenChanges))
	for _, tx := range tokenChanges {
		var messages []map[string]interface{}
		if err := json.Unmarshal(tx.Transaction.Messages, &messages); err != nil {
			logger.Get().Error().Err(err).Msg("Failed to unmarshal transaction messages")
			continue
		}

		msgTypes := make([]dto.MessageType, 0, len(messages))
		for _, msg := range messages {
			// NOTE: the original storing of message type and details
			if typeStr, ok := msg["type"].(string); ok {
				msgTypes = append(msgTypes, dto.MessageType{
					Type: typeStr,
				})
			}

			// NOTE: the new flatten message type
			if typeStr, ok := msg["@type"].(string); ok {
				msgTypes = append(msgTypes, dto.MessageType{
					Type: typeStr,
				})
			}
		}

		var tokens []map[string]interface{}
		if err := json.Unmarshal(tx.Tokens, &tokens); err != nil {
			logger.Get().Error().Err(err).Msg("Failed to unmarshal token changes")
			continue
		}

		tokenCoins := make(dto.Coins, 0, len(tokens))
		for _, token := range tokens {
			amount, amountOk := token["amount"].(string)
			denom, denomOk := token["denom"].(string)
			if amountOk && denomOk {
				tokenCoins = append(tokenCoins, dto.Coin{
					Amount: amount,
					Denom:  denom,
				})
			}
		}

		items = append(items, dto.ValidatorDelegationRelatedTx{
			Height:    int(tx.BlockHeight),
			Messages:  msgTypes,
			Sender:    tx.Transaction.Sender,
			Timestamp: tx.Block.Timestamp,
			Tokens:    tokenCoins,
			TxHash:    fmt.Sprintf("%x", tx.Transaction.Hash),
		})
	}

	return &dto.ValidatorDelegationRelatedTxsResponse{
		ValidatorDelegationRelatedTxs: items,
		Pagination:                    dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}, nil
}

func (s *validatorService) GetValidatorProposedBlocks(pagination dto.PaginationQuery, operatorAddr string) (*dto.ValidatorProposedBlocksResponse, error) {
	blocks, total, err := s.repo.GetValidatorProposedBlocks(pagination, operatorAddr)
	if err != nil {
		return nil, err
	}

	return &dto.ValidatorProposedBlocksResponse{
		ValidatorProposedBlocks: blocks,
		Pagination:              dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}, nil
}

func (s *validatorService) GetValidatorHistoricalPowers(operatorAddr string) (*dto.ValidatorHistoricalPowersResponse, error) {
	powers, total, err := s.repo.GetValidatorHistoricalPowers(operatorAddr)
	if err != nil {
		return nil, err
	}

	return &dto.ValidatorHistoricalPowersResponse{
		ValidatorHistoricalPowers: powers,
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   fmt.Sprintf("%d", total),
		},
	}, nil
}

func (s *validatorService) GetValidatorVotedProposals(pagination dto.PaginationQuery, operatorAddr, search, answer string) (*dto.ValidatorVotedProposalsResponse, error) {
	allProposals, err := s.proposalRepo.GetProposals(&pagination)
	if err != nil {
		return nil, err
	}

	validatorVotes, err := s.proposalRepo.GetProposalVotesByValidator(operatorAddr)
	if err != nil {
		return nil, err
	}

	voteMap := make(map[int32]db.ProposalVote)
	for _, vote := range validatorVotes {
		voteMap[vote.ProposalID] = vote
	}

	var filteredProposals []dto.ValidatorVotedProposal
	for _, proposal := range allProposals {
		proposalItem := dto.ValidatorVotedProposal{
			Abstain:        0,
			IsEmergency:    proposal.IsEmergency,
			IsExpedited:    proposal.IsExpedited,
			IsVoteWeighted: false,
			No:             0,
			NoWithVeto:     0,
			ProposalId:     int(proposal.ID),
			Status:         proposal.Status,
			Title:          proposal.Title,
			Yes:            0,
		}

		if proposal.Types != nil {
			var types []string
			if err := json.Unmarshal(proposal.Types, &types); err == nil {
				proposalItem.Types = types
			}
		}

		if vote, exists := voteMap[proposal.ID]; exists {
			proposalItem.Yes = vote.Yes
			proposalItem.No = vote.No
			proposalItem.NoWithVeto = vote.NoWithVeto
			proposalItem.Abstain = vote.Abstain
			proposalItem.IsVoteWeighted = vote.IsVoteWeighted
			proposalItem.Timestamp = &vote.Transaction.Block.Timestamp
			proposalItem.TxHash = fmt.Sprintf("%x", vote.Transaction.Hash)
		}

		if search != "" {
			proposalIdStr := strconv.Itoa(proposalItem.ProposalId)
			titleLower := strings.ToLower(proposalItem.Title)
			searchLower := strings.ToLower(search)

			if !strings.Contains(proposalIdStr, searchLower) && !strings.Contains(titleLower, searchLower) {
				continue
			}
		}

		if answer != "" && !matchAnswer(proposalItem, answer) {
			continue
		}

		filteredProposals = append(filteredProposals, proposalItem)
	}

	var total int64 = 0
	if pagination.CountTotal {
		total = int64(len(filteredProposals))
	}

	startIdx := pagination.Offset
	endIdx := pagination.Offset + pagination.Limit

	if startIdx > len(filteredProposals) {
		startIdx = len(filteredProposals)
	}
	if endIdx > len(filteredProposals) {
		endIdx = len(filteredProposals)
	}

	return &dto.ValidatorVotedProposalsResponse{
		ValidatorVotedProposals: filteredProposals[startIdx:endIdx],
		Pagination:              dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}, nil
}

func matchAnswer(proposal dto.ValidatorVotedProposal, answer string) bool {
	switch answer {
	case "yes":
		return proposal.Yes > 0
	case "no":
		return proposal.No > 0
	case "abstain":
		return proposal.Abstain > 0
	case "no_with_veto":
		return proposal.NoWithVeto > 0
	case "did_not_vote":
		return proposal.TxHash == ""
	case "weighted":
		return proposal.IsVoteWeighted
	default:
		return true
	}
}

func (s *validatorService) GetValidatorAnswerCounts(operatorAddr string) (*dto.ValidatorAnswerCountsResponse, error) {
	allProposals, err := s.proposalRepo.GetProposals(nil)
	if err != nil {
		return nil, err
	}

	validatorVotes, err := s.proposalRepo.GetProposalVotesByValidator(operatorAddr)
	if err != nil {
		return nil, err
	}

	allProposalsCount := len(allProposals)
	votedProposalsCount := len(validatorVotes)
	answerCounts := &dto.ValidatorAnswerCountsResponse{
		Abstain:    0,
		All:        allProposalsCount,
		DidNotVote: allProposalsCount - votedProposalsCount,
		No:         0,
		NoWithVeto: 0,
		Weighted:   0,
		Yes:        0,
	}

	for _, vote := range validatorVotes {
		if vote.IsVoteWeighted {
			answerCounts.Weighted++
			continue
		}
		if vote.Yes > 0 {
			answerCounts.Yes++
		}
		if vote.No > 0 {
			answerCounts.No++
		}
		if vote.NoWithVeto > 0 {
			answerCounts.NoWithVeto++
		}
		if vote.Abstain > 0 {
			answerCounts.Abstain++
		}
	}

	return answerCounts, nil
}
