package services

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

type TxService interface {
	GetTxByHash(hash string) (*dto.RestTxResponse, error)
	GetTxCount() (*dto.RestTxCountResponse, error)
}

type txService struct {
	repo repositories.TxRepository
}

func NewTxService(repo repositories.TxRepository) TxService {
	return &txService{
		repo: repo,
	}
}

// GetTxByHash retrieves a transaction by hash
func (s *txService) GetTxByHash(hash string) (*dto.RestTxResponse, error) {
	tx, err := s.repo.GetTxByHash(hash)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *txService) GetTxCount() (*dto.RestTxCountResponse, error) {
	txCount, err := s.repo.GetTxCount()
	if err != nil {
		return nil, err
	}

	return &dto.RestTxCountResponse{
		Count: *txCount,
	}, nil
}
