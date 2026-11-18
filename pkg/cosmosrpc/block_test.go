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
	result  *coretypes.ResultBlock
	results *coretypes.ResultBlockResults
	calls   int
}

func (c *blockClient) Block(context.Context, *int64) (*coretypes.ResultBlock, error) {
	c.calls++
	return c.result, nil
}

func (c *blockClient) BlockResults(context.Context, *int64) (*coretypes.ResultBlockResults, error) {
	return c.results, nil
}

func TestWithBlockResultsRejectsWrongHeights(t *testing.T) {
	const height int64 = 7
	for _, tc := range []struct {
		name                       string
		blockHeight, resultsHeight int64
	}{
		{"lower block", 6, 7},
		{"higher block", 8, 7},
		{"lower results", 7, 6},
		{"higher results", 7, 8},
		{"both higher", 8, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := &blockClient{
				result:  &coretypes.ResultBlock{Block: &types.Block{Header: types.Header{Height: tc.blockHeight}}},
				results: &coretypes.ResultBlockResults{Height: tc.resultsHeight},
			}
			good := &blockClient{
				result:  &coretypes.ResultBlock{Block: &types.Block{Header: types.Header{Height: height}}},
				results: &coretypes.ResultBlockResults{Height: height},
			}
			hub := &Hub{activeClients: []ActiveClient{{Client: bad}, {Client: good}}, timeout: time.Second}
			validated := 0
			err := hub.WithBlockResults(context.Background(), height, func(block *coretypes.ResultBlock, results *coretypes.ResultBlockResults) error {
				validated++
				if block != good.result || results != good.results {
					t.Errorf("wrong pair reached serializer: block=%d results=%d", block.Block.Height, results.Height)
				}
				return nil
			})
			if err != nil || validated != 1 || bad.calls != 1 || good.calls != 1 {
				t.Fatalf("failover failed: err=%v validations=%d calls=%d/%d", err, validated, bad.calls, good.calls)
			}

			hub.activeClients = []ActiveClient{{Client: bad}}
			err = hub.WithBlockResults(context.Background(), height, func(*coretypes.ResultBlock, *coretypes.ResultBlockResults) error {
				t.Error("wrong height accepted without a healthy endpoint")
				return nil
			})
			if err == nil {
				t.Fatal("expected error when no endpoint returns the requested height")
			}
		})
	}
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
