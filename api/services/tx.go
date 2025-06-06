package services

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

type TxService interface {
	GetTxByHash(hash string) (*dto.TxByHashResponse, error)
	GetTxCount() (*dto.TxCountResponse, error)
	GetTxs(pagination dto.PaginationQuery) (*dto.TxsResponse, error)
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
func (s *txService) GetTxByHash(hash string) (*dto.TxByHashResponse, error) {
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

func (s *txService) GetTxs(pagination dto.PaginationQuery) (*dto.TxsResponse, error) {
	txs, total, err := s.repo.GetTxs(pagination)
	if err != nil {
		return nil, err
	}

	response := &dto.TxsResponse{
		Txs: txs,
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}

	return response, nil
}
