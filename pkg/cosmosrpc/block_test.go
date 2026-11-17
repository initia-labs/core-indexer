package cosmosrpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
)

type blockClient struct {
	CosmosJSONRPCClient
	result *coretypes.ResultBlock
	calls  int
}

func (c *blockClient) Block(context.Context, *int64) (*coretypes.ResultBlock, error) {
	c.calls++
	return c.result, nil
}

func TestHubBlockFailsOverMissingBlock(t *testing.T) {
	for _, result := range []*coretypes.ResultBlock{nil, {}} {
		bad := &blockClient{result: result}
		good := &blockClient{result: &coretypes.ResultBlock{Block: &types.Block{Header: types.Header{Height: 7}}}}
		hub := &Hub{activeClients: []ActiveClient{{Client: bad}, {Client: good}}, timeout: time.Second}
		height := int64(7)
		got, err := hub.Block(context.Background(), &height)
		if err != nil || got != good.result || bad.calls != 1 || good.calls != 1 {
			t.Fatalf("failover failed: got=%v err=%v calls=%d/%d", got, err, bad.calls, good.calls)
		}
	}
}

func TestHubBlockMalformedRPC(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID json.RawMessage `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": map[string]any{"block": nil}})
	}))
	defer server.Close()
	hub := &Hub{activeClients: []ActiveClient{{Client: NewClient(server.URL)}}, timeout: time.Second}
	height := int64(7)
	if _, err := hub.Block(context.Background(), &height); err == nil {
		t.Fatal("expected malformed RPC error")
	}
}
