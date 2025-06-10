package services

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/utils"
)

type BlockService interface {
	GetBlockHeightLatest() (*dto.BlockHeightLatestResponse, error)
	GetBlockTimeAverage() (*dto.BlockTimeAverageResponse, error)
	GetBlocks(pagination dto.PaginationQuery) (*dto.BlocksResponse, error)
	GetBlockInfo(height int64) (*dto.BlockInfoResponse, error)
	GetBlockTxs(pagination dto.PaginationQuery, height int64) (*dto.BlockTxsResponse, error)
}

type blockService struct {
	repo repositories.BlockRepository
}

func NewBlockService(repo repositories.BlockRepository) BlockService {
	return &blockService{
		repo: repo,
	}
}

func (s *blockService) GetBlockHeightLatest() (*dto.BlockHeightLatestResponse, error) {
	height, err := s.repo.GetBlockHeightLatest()
	if err != nil {
		return nil, err
	}

	return &dto.BlockHeightLatestResponse{
		Height: *height,
	}, nil
}

func (s *blockService) GetBlockTimeAverage() (*dto.BlockTimeAverageResponse, error) {
	latestHeight, err := s.repo.GetBlockHeightLatest()
	if err != nil {
		return nil, err
	}

	if latestHeight == nil {
		return nil, nil
	}

	timestamps, err := s.repo.GetBlockTimestamp(*latestHeight)
	if err != nil {
		return nil, err
	}

	if len(timestamps) < 2 {
		return nil, nil
	}

	timeDiffs := make([]float64, 0, len(timestamps)-1)
	for idx := 0; idx < len(timestamps)-1; idx++ {
		diff := timestamps[idx].Sub(timestamps[idx+1]).Seconds()
		if diff < 0 {
			diff = -diff
		}
		timeDiffs = append(timeDiffs, diff)
	}

	medianVal := &dto.BlockTimeAverageResponse{
		AverageBlockTime: utils.Median(timeDiffs),
	}

	return medianVal, nil
}

func (s *blockService) GetBlocks(pagination dto.PaginationQuery) (*dto.BlocksResponse, error) {
	foundBlocks, total, err := s.repo.GetBlocks(pagination)
	if err != nil {
		return nil, err
	}

	blocks := make([]dto.BlockResponse, len(foundBlocks))

	for idx, block := range foundBlocks {
		blocks[idx] = dto.BlockResponse{
			Hash:      block.Hash,
			Height:    block.Height,
			Timestamp: block.Timestamp,
			TxCount:   block.TxCount,
			Proposer: dto.BlockProposerResponse{
				Identify:        block.Identity,
				Moniker:         block.Moniker,
				OperatorAddress: block.OperatorAddress,
			},
		}
	}

	return &dto.BlocksResponse{
		Blocks: blocks,
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}, nil
}

func (s *blockService) GetBlockInfo(height int64) (*dto.BlockInfoResponse, error) {
	block, err := s.repo.GetBlockInfo(height)
	if err != nil {
		return nil, err
	}

	return &dto.BlockInfoResponse{
		GasLimit:  block.GasLimit,
		GasUsed:   block.GasUsed,
		Hash:      block.Hash,
		Height:    block.Height,
		Timestamp: block.Timestamp,
		Proposer: dto.BlockProposerResponse{
			Identify:        block.Identity,
			Moniker:         block.Moniker,
			OperatorAddress: block.OperatorAddress,
		},
	}, nil
}

func (s *blockService) GetBlockTxs(pagination dto.PaginationQuery, height int64) (*dto.BlockTxsResponse, error) {
	txs, total, err := s.repo.GetBlockTxs(pagination, height)
	if err != nil {
		return nil, err
	}

	return &dto.BlockTxsResponse{
		BlockTxs: txs,
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}, nil
}
