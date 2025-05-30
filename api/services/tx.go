package services

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

type TxService interface {
	GetTxByHash(hash string) (*dto.RestTxByHashResponse, error)
	GetTxCount() (*dto.RestTxCountResponse, error)
	GetTxs(pagination dto.PaginationQuery) (*dto.RestTxsResponse, error)
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

	return txCount, nil
}

func (s *txService) GetTxs(pagination dto.PaginationQuery) (*dto.RestTxsResponse, error) {
	txs, total, err := s.repo.GetTxs(pagination)
	if err != nil {
		return nil, err
	}

	response := &dto.RestTxsResponse{
		Txs: txs,
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}

	// If we have items and count_total is true, we can calculate the next key
	if len(txs) > 0 && pagination.CountTotal {
		// TODO: Implement next key calculation based on the last item
		// This would typically be a base64 encoded cursor to the next page
	}

	return response, nil
}
