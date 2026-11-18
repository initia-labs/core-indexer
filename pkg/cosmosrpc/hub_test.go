package cosmosrpc

import (
	"context"
	"errors"
	"testing"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/rs/zerolog"
)

type refreshClient struct {
	CosmosJSONRPCClient
	statusErr error
	height    int64
}

func (c *refreshClient) Status(context.Context) (*coretypes.ResultStatus, error) {
	if c.statusErr != nil {
		return nil, c.statusErr
	}
	result := &coretypes.ResultStatus{}
	result.SyncInfo.LatestBlockHeight = c.height
	return result, nil
}
func (c *refreshClient) GetIdentifier() string { return "test" }

func TestRebalancePreservesSnapshotOnFailure(t *testing.T) {
	client := &refreshClient{height: 10}
	logger := zerolog.Nop()
	hub := &Hub{Clients: []CosmosJSONRPCClient{client}, logger: &logger, timeout: time.Second}
	ctx := context.Background()
	if err := hub.Rebalance(ctx); err != nil {
		t.Fatal(err)
	}
	client.statusErr = errors.New("status unavailable")
	if err := hub.Rebalance(ctx); err == nil {
		t.Fatal("expected refresh failure")
	}
	active := hub.GetActiveClients()
	if len(active) != 1 || active[0].Client != client || active[0].Height != 10 {
		t.Fatalf("lost previous snapshot: %+v", active)
	}
	client.statusErr = nil
	client.height = 20
	result, err := hub.Status(ctx)
	if err != nil || result.SyncInfo.LatestBlockHeight != 20 {
		t.Fatalf("retained endpoint unusable: result=%v err=%v", result, err)
	}
	if err := hub.Rebalance(ctx); err != nil {
		t.Fatal(err)
	}
	if got := hub.GetActiveClients()[0].Height; got != 20 {
		t.Fatalf("snapshot height = %d, want 20", got)
	}
}
