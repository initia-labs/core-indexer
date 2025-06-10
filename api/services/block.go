package services

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/utils"
)

type BlockService interface {
	GetBlockHeightLatest() (*dto.BlockHeightLatestResponse, error)
	GetBlockTimeAverage() (*dto.BlockTimeAverageResponse, error)
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
