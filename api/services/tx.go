package services

import (
	"fmt"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

type TxService interface {
	GetTxByHash(hash string) (*dto.TxByHashResponse, error)
	GetTxCount() (*dto.TxCountResponse, error)
	GetTxs(pagination dto.PaginationQuery) (*dto.TxsResponse, error)
}

type txService struct {
	repo repositories.TxRepositoryI
}

func NewTxService(repo repositories.TxRepositoryI) TxService {
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
		Txs: make([]dto.TxModel, len(txs)),
		Pagination: dto.PaginationResponse{
			NextKey: nil,
			Total:   total,
		},
	}

	for idx, tx := range txs {
		response.Txs[idx] = dto.TxModel{
			Sender:    tx.Sender,
			Hash:      fmt.Sprintf("%x", tx.Hash),
			Success:   tx.Success,
			Messages:  tx.Messages,
			IsSend:    tx.IsSend,
			IsIbc:     tx.IsIbc,
			IsOpinit:  tx.IsOpinit,
			Height:    tx.Height,
			Timestamp: tx.Timestamp,
		}
	}

	return response, nil
}
