package cosmosrpc

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	initiagovtypes "github.com/initia-labs/initia/x/gov/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	"github.com/rs/zerolog"

	"github.com/initia-labs/core-indexer/pkg/sentry_integration"
)

type ClientConfig struct {
	URL          string
	ClientOption *ClientOption
}

type Hub struct {
	mu            sync.Mutex
	Clients       []CosmosJSONRPCClient
	activeClients []ActiveClient
	logger        *zerolog.Logger
	timeout       time.Duration
}

type ActiveClient struct {
	Client CosmosJSONRPCClient
	Height int64
}

type CosmosJSONRPCHub interface {
	CosmosJSONRPCClient
	Rebalance(ctx context.Context) error
	GetActiveClients() []ActiveClient
}

func NewHub(configs []ClientConfig, logger *zerolog.Logger, timeout time.Duration) *Hub {
	clients := make([]CosmosJSONRPCClient, 0)
	for _, config := range configs {
		var client *Client
		if config.ClientOption == nil {
			client = NewClient(config.URL)
		} else {
			client = NewClientWithOption(config.URL, *config.ClientOption)
		}
		clients = append(clients, client)
	}
	return &Hub{
		Clients: clients,
		logger:  logger,
		timeout: timeout,
	}
}

func (h *Hub) Rebalance(ctx context.Context) error {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "Rebalance", "Rebalance hub rpcs")
	defer span.Finish()

	var result *coretypes.ResultStatus
	var err error
	clients := make([]ActiveClient, 0)
	for _, client := range h.Clients {
		ctx, cancel := createTimeoutContext(ctx, h.timeout)
		defer cancel()
		result, err = client.Status(ctx)
		if err != nil {
			err = handleTimeoutError(err)
			h.logger.Error().Err(err).Msgf("Failed to get client from id :%s status: %s", client.GetIdentifier(), err)
			continue
		}
		clients = append(clients, ActiveClient{Client: client, Height: result.SyncInfo.LatestBlockHeight})
	}

	sort.Slice(clients, func(i, j int) bool {
		return clients[i].Height >= clients[j].Height
	})

	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeClients = clients
	if len(h.activeClients) == 0 {
		return fmt.Errorf("RPC: All RPC Clients failed to get status. Last error: %v", err)
	}

	return nil
}

func (h *Hub) Status(ctx context.Context) (*coretypes.ResultStatus, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "HubStatus", "Calling /status from RPCs")
	defer span.Finish()

	result, err := handleQuery(ctx, h.timeout, h.activeClients, func(ctx context.Context, c ActiveClient) (*coretypes.ResultStatus, error) {
		return c.Client.Status(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %v", err)
	}

	return result, nil
}

func (h *Hub) Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "HubBlock", "Calling /block from RPCs")
	defer span.Finish()

	result, err := handleQuery(ctx, h.timeout, h.activeClients, func(ctx context.Context, c ActiveClient) (*coretypes.ResultBlock, error) {
		result, err := c.Client.Block(ctx, height)
		if err != nil {
			return nil, err
		} else if result.Block.Header.Height < *height {
			return nil, fmt.Errorf("RPC: Stale block data")
		} else {
			return result, nil
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %v", err)
	}

	return result, nil
}

func (h *Hub) BlockResults(ctx context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "HubBlockResults", "Calling /block_results from RPCs")
	defer span.Finish()

	result, err := handleQuery(ctx, h.timeout, h.activeClients, func(ctx context.Context, c ActiveClient) (*coretypes.ResultBlockResults, error) {
		result, err := c.Client.BlockResults(ctx, height)
		if err != nil {
			return nil, err
		} else if result.Height < *height {
			return nil, fmt.Errorf("RPC: Stale block results data")
		} else {
			return result, nil
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get block results: %v", err)
	}

	return result, nil
}

func (h *Hub) Proposal(ctx context.Context, proposalID int32, height *int64) (*initiagovtypes.QueryProposalResponse, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "HubProposal", "Calling /proposal from RPCs")
	defer span.Finish()

	result, err := handleQuery(ctx, h.timeout, h.activeClients, func(ctx context.Context, c ActiveClient) (*initiagovtypes.QueryProposalResponse, error) {
		return c.Client.Proposal(ctx, proposalID, height)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get proposal: %v", err)
	}

	return result, nil
}

func (h *Hub) Validator(ctx context.Context, validatorAddress string, height *int64) (*mstakingtypes.QueryValidatorResponse, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "HubValidator", "Calling /validator from RPCs")
	defer span.Finish()

	result, err := handleQuery(ctx, h.timeout, h.activeClients, func(ctx context.Context, c ActiveClient) (*mstakingtypes.QueryValidatorResponse, error) {
		return c.Client.Validator(ctx, validatorAddress, height)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get validator: %v", err)
	}

	return result, nil
}

func (h *Hub) Validators(ctx context.Context, status string, height *int64) (*[]mstakingtypes.Validator, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "HubValidatorInfos", "Calling validator infos from RPCs")
	defer span.Finish()

	result, err := handleQuery(ctx, h.timeout, h.activeClients, func(ctx context.Context, c ActiveClient) (*[]mstakingtypes.Validator, error) {
		return c.Client.Validators(ctx, status, height)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get validators: %v", err)
	}

	return result, nil
}

func (h *Hub) Module(ctx context.Context, address, moduleName string, height *int64) (*movetypes.QueryModuleResponse, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "HubModule", "Calling /module from RPCs")
	defer span.Finish()

	result, err := handleQuery(ctx, h.timeout, h.activeClients, func(ctx context.Context, c ActiveClient) (*movetypes.QueryModuleResponse, error) {
		return c.Client.Module(ctx, address, moduleName, height)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get module: %v", err)
	}

	return result, nil
}

func (h *Hub) Resource(ctx context.Context, address, structTag string, height *int64) (*movetypes.QueryResourceResponse, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "HubResource", "Calling /resource from RPCs")
	defer span.Finish()

	result, err := handleQuery(ctx, h.timeout, h.activeClients, func(ctx context.Context, c ActiveClient) (*movetypes.QueryResourceResponse, error) {
		return c.Client.Resource(ctx, address, structTag, height)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get module resource: %v", err)
	}

	return result, nil
}

func (h *Hub) Genesis(ctx context.Context) (*coretypes.ResultGenesis, error) {
	span, ctx := sentry_integration.StartSentrySpan(ctx, "HubGenesis", "Calling /genesis from RPCs")
	defer span.Finish()

	result, err := handleQuery(ctx, h.timeout, h.activeClients, func(ctx context.Context, c ActiveClient) (*coretypes.ResultGenesis, error) {
		return c.Client.Genesis(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get genesis: %v", err)
	}

	return result, nil
}

func (h *Hub) GetActiveClients() []ActiveClient {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.activeClients
}

func (h *Hub) GetIdentifier() string {
	if len(h.Clients) == 0 {
		return ""
	}

	ids := make([]string, len(h.Clients))
	for i, client := range h.Clients {
		ids[i] = client.GetIdentifier()
	}
	return fmt.Sprintf("%v", ids)
}
