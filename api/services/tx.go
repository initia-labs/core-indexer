package services

import "github.com/initia-labs/core-indexer/api/repositories"

type TxService interface {
	GetTxCount() (*int64, error)
}

type txService struct {
	repo repositories.TxRepository
}

func NewTxService(repo repositories.TxRepository) TxService {
	return &txService{
		repo: repo,
	}
}

func (s *txService) GetTxCount() (*int64, error) {
	txCount, err := s.repo.GetTxCount()
	if err != nil {
		return nil, err
	}

	return txCount, nil
}
