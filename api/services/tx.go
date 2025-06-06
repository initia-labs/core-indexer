package services

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

type TxService interface {
	GetTxByHash(hash string) (*dto.RestTxByHashResponse, error)
	GetTxCount() (*dto.RestTxCountResponse, error)
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
func (s *txService) GetTxByHash(hash string) (*dto.RestTxByHashResponse, error) {
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
