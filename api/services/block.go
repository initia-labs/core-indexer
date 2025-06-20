package services

import (
	"fmt"

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
	repo repositories.BlockRepositoryI
}

func NewBlockService(repo repositories.BlockRepositoryI) BlockService {
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

	blocks := make([]dto.Block, len(foundBlocks))

	for idx, block := range foundBlocks {
		blocks[idx] = dto.Block{
			Hash:      fmt.Sprintf("%x", block.Hash),
			Height:    block.Height,
			Timestamp: block.Timestamp,
			TxCount:   block.TxCount,
			Proposer: dto.BlockProposer{
				Identity:        block.Identity,
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
		Hash:      fmt.Sprintf("%x", block.Hash),
		Height:    block.Height,
		Timestamp: block.Timestamp,
		Proposer: dto.BlockProposer{
			Identity:        block.Identity,
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

	blockTxs := make([]dto.BlockTxModel, len(txs))
	for idx, tx := range txs {
		blockTxs[idx] = dto.BlockTxModel{
			Height:    tx.Height,
			Timestamp: tx.Timestamp,
			Address:   tx.Address,
			Hash:      fmt.Sprintf("%x", tx.Hash),
			Success:   tx.Success,
			Messages:  tx.Messages,
			IsSend:    tx.IsSend,
			IsIbc:     tx.IsIbc,
			IsOpinit:  tx.IsOpinit,
		}
	}

	return &dto.BlockTxsResponse{
		BlockTxs: blockTxs,
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}, nil
}
