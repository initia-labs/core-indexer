package cosmosrpc

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	cjson "github.com/cometbft/cometbft/libs/json"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/cosmos-sdk/types/query"
	movetypes "github.com/initia-labs/initia/x/move/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	"github.com/ybbus/jsonrpc/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
)

type Client struct {
	jc         jsonrpc.RPCClient
	clientCtx  client.Context
	identifier string
}

func generateHeader(height *int64) *metadata.MD {
	header := metadata.New(map[string]string{})
	if height != nil {
		header.Append(grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(*height, 10))
	}
	return &header
}

func (c *Client) Call(ctx context.Context, method string, params map[string]any) (*jsonrpc.RPCResponse, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, c.identifier+"/"+method, "Calling "+method+" of "+c.identifier)
	defer span.Finish()

	paramsMap := make(map[string]json.RawMessage, len(params))
	for name, value := range params {
		valueJSON, err := cjson.Marshal(value)
		if err != nil {
			return nil, err
		}
		paramsMap[name] = valueJSON
	}

	return c.jc.Call(ctx, method, paramsMap)
}

type CosmosJSONRPCClient interface {
	Status(ctx context.Context) (*coretypes.ResultStatus, error)
	Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error)
	BlockResults(ctx context.Context, height *int64) (*coretypes.ResultBlockResults, error)
	Validator(ctx context.Context, validatorAddress string, height *int64) (*mstakingtypes.QueryValidatorResponse, error)
	Validators(ctx context.Context, height *int64, page, perPage *int) (*coretypes.ResultValidators, error)
	ValidatorInfos(ctx context.Context, status string, height *int64) (*[]mstakingtypes.Validator, error)
	Module(ctx context.Context, address, moduleName string, height *int64) (*movetypes.QueryModuleResponse, error)
	GetIdentifier() string
}

type ClientOption struct {
	HTTPClient         *http.Client
	CustomHeaders      map[string]string
	AllowUnknownFields bool
	DefaultRequestID   int
}

func NewClient(url string) *Client {
	c, err := client.NewClientFromNode(url)
	if err != nil {
		panic(err)
	}
	return &Client{jsonrpc.NewClient(url), client.Context{}.WithClient(c), url}
}

func NewClientWithOption(url string, option ClientOption) *Client {
	c, err := client.NewClientFromNode(url)
	if err != nil {
		panic(err)
	}
	return &Client{
		jsonrpc.NewClientWithOpts(url, &jsonrpc.RPCClientOpts{
			HTTPClient:         option.HTTPClient,
			CustomHeaders:      option.CustomHeaders,
			AllowUnknownFields: option.AllowUnknownFields,
			DefaultRequestID:   option.DefaultRequestID,
		}),
		client.Context{}.WithClient(c),
		url,
	}
}

func handleResponseAndGetResult[T any](response *jsonrpc.RPCResponse, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, response.Error
	}

	jsBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, err
	}

	result := new(T)
	err = cjson.Unmarshal(jsBytes, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) Status(ctx context.Context) (*coretypes.ResultStatus, error) {
	jsonResponse, err := c.Call(ctx, "status", map[string]any{})
	return handleResponseAndGetResult[coretypes.ResultStatus](jsonResponse, err)
}

func (c *Client) Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error) {
	jsonResponse, err := c.Call(ctx, "block", map[string]any{"height": height})
	return handleResponseAndGetResult[coretypes.ResultBlock](jsonResponse, err)
}

func (c *Client) BlockResults(ctx context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
	jsonResponse, err := c.Call(ctx, "block_results", map[string]any{"height": height})
	return handleResponseAndGetResult[coretypes.ResultBlockResults](jsonResponse, err)
}

func (c *Client) Validators(ctx context.Context, height *int64, page, perPage *int) (*coretypes.ResultValidators, error) {
	params := make(map[string]any)
	if page != nil {
		params["page"] = page
	}
	if perPage != nil {
		params["per_page"] = perPage
	}
	if height != nil {
		params["height"] = height
	}
	jsonResponse, err := c.Call(ctx, "validators", params)
	return handleResponseAndGetResult[coretypes.ResultValidators](jsonResponse, err)
}

func (c *Client) ValidatorInfos(ctx context.Context, status string, height *int64) (*[]mstakingtypes.Validator, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, c.identifier+"/validator_infos", "Calling validator_infos of "+c.identifier)
	defer span.Finish()

	queryClient := mstakingtypes.NewQueryClient(c.clientCtx)
	nextKey := make([]byte, 0)
	vals := make([]mstakingtypes.Validator, 0)
	for {
		request := mstakingtypes.QueryValidatorsRequest{
			Pagination: &query.PageRequest{
				Key: nextKey,
			},
		}
		if status != "" {
			request.Status = status
		}
		result, err := queryClient.Validators(ctx, &request, grpc.Header(generateHeader(height)))
		if err != nil {
			return nil, err
		}
		nextKey = result.Pagination.NextKey
		vals = append(vals, result.Validators...)
		if len(nextKey) == 0 {
			break
		}
	}
	return &vals, nil
}

func (c *Client) Module(ctx context.Context, address, moduleName string, height *int64) (*movetypes.QueryModuleResponse, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, c.identifier+"/module", "Calling module of "+c.identifier)
	defer span.Finish()

	queryClient := movetypes.NewQueryClient(c.clientCtx)
	request := movetypes.QueryModuleRequest{
		Address:    address,
		ModuleName: moduleName,
	}
	header := metadata.New(map[string]string{})
	if height != nil {
		header.Append(grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(*height, 10))
	}
	result, err := queryClient.Module(ctx, &request, grpc.Header(&header))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Validator(ctx context.Context, ValidatorAddr string, height *int64) (*mstakingtypes.QueryValidatorResponse, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, c.identifier+"/validator", "Calling validator of "+c.identifier)
	defer span.Finish()

	queryClient := mstakingtypes.NewQueryClient(c.clientCtx)
	request := mstakingtypes.QueryValidatorRequest{
		ValidatorAddr: ValidatorAddr,
	}

	result, err := queryClient.Validator(ctx, &request, grpc.Header(generateHeader(height)))
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) GetIdentifier() string {
	return c.identifier
}
