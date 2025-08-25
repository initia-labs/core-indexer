package mocks

import (
	"time"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/pkg/db"
	"github.com/stretchr/testify/mock"
)

type BlockRepository struct {
	mock.Mock
}

func NewMockBlockRepository() *BlockRepository {
	return &BlockRepository{}
}

func (m *BlockRepository) GetBlockHeightLatest() (*int64, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*int64), args.Error(1)
}

func (m *BlockRepository) GetBlockHeightInformativeLatest() (*int64, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*int64), args.Error(1)
}

func (m *BlockRepository) GetBlockTimestamp(latestBlockHeight int64) ([]time.Time, error) {
	args := m.Called(latestBlockHeight)
	return args.Get(0).([]time.Time), args.Error(1)
}

func (m *BlockRepository) GetBlocks(pagination dto.PaginationQuery) ([]dto.BlockModel, int64, error) {
	args := m.Called(pagination)
	return args.Get(0).([]dto.BlockModel), args.Get(1).(int64), args.Error(2)
}

func (m *BlockRepository) GetBlockInfo(height int64) (*dto.BlockInfoModel, error) {
	args := m.Called(height)
	return args.Get(0).(*dto.BlockInfoModel), args.Error(1)
}

func (m *BlockRepository) GetBlockTxs(pagination dto.PaginationQuery, height int64) ([]dto.BlockTxModel, int64, error) {
	args := m.Called(pagination, height)
	return args.Get(0).([]dto.BlockTxModel), args.Get(1).(int64), args.Error(2)
}

func (m *BlockRepository) GetLatestBlock() (*db.Block, error) {
	args := m.Called()
	return args.Get(0).(*db.Block), args.Error(1)
}
