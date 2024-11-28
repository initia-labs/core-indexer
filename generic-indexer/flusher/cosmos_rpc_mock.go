package flusher

import (
	"context"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/initia-labs/initia/x/mstaking/types"
	"github.com/stretchr/testify/mock"
	"github.com/ybbus/jsonrpc/v3"

	"github.com/initia-labs/core-indexer/generic-indexer/cosmosrpc"
)

var _ cosmosrpc.CosmosJSONRPCClient = &MockClient{}

type MockClient struct {
	jsonrpc.RPCClient
	mock.Mock
}

func (c *MockClient) Status(ctx context.Context) (*coretypes.ResultStatus, error) {
	return &coretypes.ResultStatus{}, nil
}

func (c *MockClient) Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error) {
	return &coretypes.ResultBlock{}, nil
}

func (c *MockClient) BlockResults(ctx context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
	return &coretypes.ResultBlockResults{}, nil
}

func (c *MockClient) Validators(ctx context.Context, height *int64, page, perPage *int) (*coretypes.ResultValidators, error) {
	args := c.Called(ctx, height, page, perPage)
	return args.Get(0).(*coretypes.ResultValidators), args.Error(1)
}

func (c *MockClient) ValidatorInfos(ctx context.Context, status string) (*[]types.Validator, error) {
	args := c.Called(ctx)
	return args.Get(0).(*[]types.Validator), args.Error(1)
}

func (c *MockClient) GetIdentifier() string {
	return ""
}

func (c *MockClient) Rebalance(ctx context.Context) error {
	return nil
}
