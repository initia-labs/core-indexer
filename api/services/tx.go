package services

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

// TxService defines the interface for transaction-related operations
type TxService interface {
	GetTxByHash(hash string) (*dto.RestTxResponse, error)
	GetTxCount() (*dto.TxCountResponse, error)
}

// txService implements the TxService interface
type txService struct {
	repo repositories.TxRepository
}

// NewTxService creates a new instance of TxService
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

// GetTxCount retrieves the total number of transactions
func (s *txService) GetTxCount() (*dto.TxCountResponse, error) {
	txCount, err := s.repo.GetTxCount()
	if err != nil {
		return nil, err
	}

	return &dto.TxCountResponse{
		Count: *txCount,
	}, nil
}
