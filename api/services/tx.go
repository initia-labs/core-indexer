package services

import (
	"context"
	"fmt"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
)

type TxService interface {
	GetTxByHash(ctx context.Context, hash string) (*dto.TxByHashResponse, error)
	GetTxCount(ctx context.Context) (*dto.TxCountResponse, error)
	GetTxs(ctx context.Context, pagination *dto.PaginationQuery) (*dto.TxsModelResponse, error)
	GetTxsByAccountAddress(ctx context.Context, pagination dto.PaginationQuery, accountAddress string) (*dto.TxsResponse, error)
}

type txService struct {
	txRepo      repositories.TxRepositoryI
	accountRepo repositories.AccountRepositoryI
	gcsManager  GCSManager
}

func NewTxService(txRepo repositories.TxRepositoryI, accountRepo repositories.AccountRepositoryI) TxService {
	return &txService{
		txRepo:      txRepo,
		accountRepo: accountRepo,
		gcsManager:  NewGCSManager(txRepo),
	}
}

// GetTxByHash retrieves a transaction by hash
func (s *txService) GetTxByHash(ctx context.Context, hash string) (*dto.TxByHashResponse, error) {
	tx, err := s.txRepo.GetTxByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// GetTxCount retrieves the total number of transactions
func (s *txService) GetTxCount(ctx context.Context) (*dto.TxCountResponse, error) {
	txCount, err := s.txRepo.GetTxCount(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.TxCountResponse{
		Count: *txCount,
	}, nil
}

func (s *txService) GetTxs(ctx context.Context, pagination *dto.PaginationQuery) (*dto.TxsModelResponse, error) {
	txs, total, err := s.txRepo.GetTxs(ctx, pagination)
	if err != nil {
		return nil, err
	}

	response := &dto.TxsModelResponse{
		Txs:        make([]dto.TxModel, len(txs)),
		Pagination: dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
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

func (s *txService) GetTxsByAccountAddress(ctx context.Context, pagination dto.PaginationQuery, accountAddress string) (*dto.TxsResponse, error) {
	txs, total, err := s.accountRepo.GetAccountTxs(ctx, pagination, accountAddress, "", false, false, false, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	response := &dto.TxsResponse{
		Txs:        make([]dto.TxResponse, len(txs)),
		Pagination: dto.NewPaginationResponse(pagination.Offset, pagination.Limit, total),
	}

	txHashes := make([]string, len(txs))
	for idx, tx := range txs {
		txHashes[idx] = fmt.Sprintf("%x", tx.Hash)
	}

	res, err := s.gcsManager.QueryTxs(ctx, txHashes)
	if err != nil {
		return nil, err
	}

	response.Txs = make([]dto.TxResponse, len(res))
	for idx, tx := range res {
		response.Txs[idx] = tx.TxResponse
	}

	return response, nil
}
