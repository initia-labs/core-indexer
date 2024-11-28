package cosmosrpc

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/initia-labs/initia/x/mstaking/types"
	"github.com/rs/zerolog"

	"github.com/initia-labs/core-indexer/generic-indexer/common"
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

// createTimeoutContext checks if the provided context already has a deadline or timeout.
// If it does, it returns the existing context and a no-op cancel function.
// If not, it creates a new context with the specified timeout.
func createTimeoutContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		// If context already has a deadline, return the existing context and a no-op cancel function.
		return ctx, func() {}
	}
	// Create a new context with the specified timeout.
	return context.WithTimeout(ctx, timeout)
}

func (h *Hub) handleTimeoutError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("RPC: request timed out")
	}
	return err
}

func (h *Hub) Rebalance(ctx context.Context) error {
	span, ctx := common.StartSentrySpan(ctx, "Rebalance", "Rebalance hub rpcs")
	defer span.Finish()

	var result *coretypes.ResultStatus
	var err error
	clients := make([]ActiveClient, 0)
	for _, client := range h.Clients {
		ctx, cancel := createTimeoutContext(ctx, h.timeout)
		defer cancel()
		result, err = client.Status(ctx)
		if err != nil {
			err = h.handleTimeoutError(err)
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
	span, ctx := common.StartSentrySpan(ctx, "HubStatus", "Calling /status from RPCs")
	defer span.Finish()

	var result *coretypes.ResultStatus
	var err error
	for _, active := range h.activeClients {
		ctx, cancel := createTimeoutContext(ctx, h.timeout)
		defer cancel()
		result, err = active.Client.Status(ctx)
		if err != nil {
			err = h.handleTimeoutError(err)
			h.logger.Error().Err(err).Msgf("Failed to get client status: %v", err)
			continue
		}
		return result, nil
	}
	return nil, fmt.Errorf("RPC: All RPC Clients failed to get status. Last error: %v", err)
}

func (h *Hub) Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error) {
	span, ctx := common.StartSentrySpan(ctx, "HubBlock", "Calling /block from RPCs")
	defer span.Finish()

	var result *coretypes.ResultBlock
	var err error
	var hasStaleData bool
	for _, active := range h.activeClients {
		ctx, cancel := createTimeoutContext(ctx, h.timeout)
		defer cancel()
		result, err = active.Client.Block(ctx, height)
		if err != nil {
			err = h.handleTimeoutError(err)
			continue
		} else if result.Block.Header.Height < *height {
			hasStaleData = true
		} else {
			return result, nil
		}
	}
	if hasStaleData {
		return nil, fmt.Errorf("RPC: Stale block data")
	}
	return nil, fmt.Errorf("RPC: All RPC Clients failed to get block. Last error: %v", err)
}

func (h *Hub) BlockResults(ctx context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
	span, ctx := common.StartSentrySpan(ctx, "HubBlockResults", "Calling /block_results from RPCs")
	defer span.Finish()

	var result *coretypes.ResultBlockResults
	var err error
	var hasStaleData bool
	for _, active := range h.activeClients {
		ctx, cancel := createTimeoutContext(ctx, h.timeout)
		defer cancel()
		result, err = active.Client.BlockResults(ctx, height)
		if err != nil {
			err = h.handleTimeoutError(err)
			h.logger.Error().Err(err).Msgf("Failed to get client status: %v", err)
			continue
		} else if result.Height < *height {
			hasStaleData = true
		} else {
			return result, nil
		}
	}

	if hasStaleData {
		return nil, fmt.Errorf("RPC: Stale block results data")
	}
	return nil, fmt.Errorf("RPC: All RPC Clients failed to get block results. Last error: %v", err)
}

func (h *Hub) Validators(ctx context.Context, height *int64, page, perPage *int) (*coretypes.ResultValidators, error) {
	span, ctx := common.StartSentrySpan(ctx, "HubValidators", "Calling /validators from RPCs")
	defer span.Finish()

	var result *coretypes.ResultValidators
	var err error
	for _, active := range h.activeClients {
		ctx, cancel := createTimeoutContext(ctx, h.timeout)
		defer cancel()
		result, err = active.Client.Validators(ctx, height, page, perPage)
		if err != nil {
			err = h.handleTimeoutError(err)
			h.logger.Error().Err(err).Msgf("Failed to get client status: %v", err)
			continue
		}
		return result, nil
	}

	return nil, fmt.Errorf("RPC: All RPC Clients failed to get validators results. Last error: %v", err)
}

func (h *Hub) ValidatorInfos(ctx context.Context, status string) (*[]types.Validator, error) {
	span, ctx := common.StartSentrySpan(ctx, "HubValidatorInfos", "Calling validator infos from RPCs")
	defer span.Finish()

	var result *[]types.Validator
	var err error
	for _, active := range h.activeClients {
		result, err = active.Client.ValidatorInfos(ctx, status)
		if err != nil {
			err = h.handleTimeoutError(err)
			h.logger.Error().Err(err).Msgf("Failed to get client status: %v", err)
			continue
		}
		return result, nil
	}
	return nil, fmt.Errorf("RPC: All RPC Clients failed to get validators results. Last error: %v", err)
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
