package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
)

type BlockRepository struct {
	mock.Mock
}

func NewMockBlockRepository() *BlockRepository {
	return &BlockRepository{}
}

func (m *BlockRepository) GetBlockHeightLatest(ctx context.Context) (*int64, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*int64), args.Error(1)
}

func (m *BlockRepository) GetBlockHeightInformativeLatest(ctx context.Context) (*int64, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*int64), args.Error(1)
}

func (m *BlockRepository) GetBlockTimestamp(ctx context.Context, latestBlockHeight int64) ([]time.Time, error) {
	args := m.Called(ctx, latestBlockHeight)
	return args.Get(0).([]time.Time), args.Error(1)
}

func (m *BlockRepository) GetBlocks(ctx context.Context, pagination dto.PaginationQuery) ([]dto.BlockModel, int64, error) {
	args := m.Called(ctx, pagination)
	return args.Get(0).([]dto.BlockModel), args.Get(1).(int64), args.Error(2)
}

func (m *BlockRepository) GetBlockInfo(ctx context.Context, height int64) (*dto.BlockInfoModel, error) {
	args := m.Called(ctx, height)
	return args.Get(0).(*dto.BlockInfoModel), args.Error(1)
}

func (m *BlockRepository) GetBlockTxs(ctx context.Context, pagination dto.PaginationQuery, height int64) ([]dto.BlockTxModel, int64, error) {
	args := m.Called(ctx, pagination, height)
	return args.Get(0).([]dto.BlockTxModel), args.Get(1).(int64), args.Error(2)
}

func (m *BlockRepository) GetLatestBlock(ctx context.Context) (*db.Block, error) {
	args := m.Called(ctx)
	return args.Get(0).(*db.Block), args.Error(1)
}
