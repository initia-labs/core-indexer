package cosmosrpc

import (
	"context"
	"encoding/json"
	"net/http"

	cjson "github.com/cometbft/cometbft/libs/json"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/query"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	"github.com/ybbus/jsonrpc/v3"

	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
)

type Client struct {
	jc         jsonrpc.RPCClient
	clientCtx  client.Context
	identifier string
}

func (c *Client) Call(ctx context.Context, method string, params map[string]interface{}) (*jsonrpc.RPCResponse, error) {
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
	Validators(ctx context.Context, height *int64, page, perPage *int) (*coretypes.ResultValidators, error)
	ValidatorInfos(ctx context.Context, status string) (*[]mstakingtypes.Validator, error)
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
	jsonResponse, err := c.Call(ctx, "status", map[string]interface{}{})
	return handleResponseAndGetResult[coretypes.ResultStatus](jsonResponse, err)
}

func (c *Client) Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error) {
	jsonResponse, err := c.Call(ctx, "block", map[string]interface{}{"height": height})
	return handleResponseAndGetResult[coretypes.ResultBlock](jsonResponse, err)
}

func (c *Client) BlockResults(ctx context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
	jsonResponse, err := c.Call(ctx, "block_results", map[string]interface{}{"height": height})
	return handleResponseAndGetResult[coretypes.ResultBlockResults](jsonResponse, err)
}

func (c *Client) Validators(ctx context.Context, height *int64, page, perPage *int) (*coretypes.ResultValidators, error) {
	params := make(map[string]interface{})
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

func (c *Client) ValidatorInfos(ctx context.Context, status string) (*[]mstakingtypes.Validator, error) {
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
		result, err := queryClient.Validators(ctx, &request)
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

func (c *Client) GetIdentifier() string {
	return c.identifier
}
