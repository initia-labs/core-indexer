package services

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/utils"
)

type BlockService interface {
	GetBlockHeightLatest() (*dto.RestBlockHeightLatestResponse, error)
	GetBlockTimeAverage() (*dto.RestBlockTimeAverageResponse, error)
}

type blockService struct {
	repo repositories.BlockRepository
}

func NewBlockService(repo repositories.BlockRepository) BlockService {
	return &blockService{
		repo: repo,
	}
}

func (s *blockService) GetBlockHeightLatest() (*dto.RestBlockHeightLatestResponse, error) {
	latestBlockHeight, err := s.repo.GetBlockHeightLatest()
	if err != nil {
		return nil, err
	}

	return latestBlockHeight, nil
}

func (s *blockService) GetBlockTimeAverage() (*dto.RestBlockTimeAverageResponse, error) {
	blockHeightLatest, err := s.repo.GetBlockHeightLatest()
	if err != nil {
		return nil, err
	}

	if blockHeightLatest == nil {
		return nil, nil
	}

	timestamps, err := s.repo.GetBlockTimestamp(&blockHeightLatest.Height)
	if err != nil {
		return nil, err
	}

	if len(timestamps) < 2 {
		return nil, nil
	}

	timeDiffs := make([]float64, 0, len(timestamps)-1)
	for i := 0; i < len(timestamps)-1; i++ {
		diff := timestamps[i].Sub(timestamps[i+1]).Seconds()
		if diff < 0 {
			diff = -diff
		}
		timeDiffs = append(timeDiffs, diff)
	}

	medianVal := &dto.RestBlockTimeAverageResponse{
		AverageBlockTime: utils.Median(timeDiffs),
	}
	return medianVal, nil
}
