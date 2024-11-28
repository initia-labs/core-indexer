package cosmosrpc

import (
	"context"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/initia-labs/initia/x/mstaking/types"
	"github.com/stretchr/testify/mock"
	"github.com/ybbus/jsonrpc/v3"
)

var _ CosmosJSONRPCClient = &MockClient{}

type MockClient struct {
	jsonrpc.RPCClient
	url string
	mock.Mock
}

func (c *MockClient) Status(ctx context.Context) (*coretypes.ResultStatus, error) {
	args := c.Called(ctx)
	return args.Get(0).(*coretypes.ResultStatus), args.Error(1)
}

func (c *MockClient) Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error) {
	args := c.Called(ctx)
	return args.Get(0).(*coretypes.ResultBlock), args.Error(1)
}

func (c *MockClient) BlockResults(ctx context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
	args := c.Called(ctx)
	return args.Get(0).(*coretypes.ResultBlockResults), args.Error(1)
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
	return c.url
}
