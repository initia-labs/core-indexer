package alerts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	coredb "github.com/initia-labs/core-indexer/pkg/db"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"github.com/rs/zerolog"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type alertRPC struct {
	cosmosrpc.CosmosJSONRPCClient
	chain      string
	head       int64
	failNext   bool
	incomplete bool
	calls      []int64
}

func (r *alertRPC) GetIdentifier() string { return "local-test" }
func (r *alertRPC) Status(context.Context) (*coretypes.ResultStatus, error) {
	return &coretypes.ResultStatus{SyncInfo: coretypes.SyncInfo{LatestBlockHeight: r.head}}, nil
}
func (r *alertRPC) Block(_ context.Context, height *int64) (*coretypes.ResultBlock, error) {
	r.calls = append(r.calls, *height)
	if r.failNext {
		r.failNext = false
		r.head += 2
		return nil, fmt.Errorf("temporary RPC failure")
	}
	return &coretypes.ResultBlock{Block: &types.Block{
		Header: types.Header{ChainID: r.chain, Height: *height, Time: time.Unix(*height, 0), ProposerAddress: make([]byte, 20)},
		Data:   types.Data{Txs: types.Txs{[]byte(fmt.Sprintf("success-%d", *height)), []byte(fmt.Sprintf("failed-%d", *height))}},
	}}, nil
}
func (r *alertRPC) BlockResults(_ context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
	e := packet("channel-2", "transfer/channel-2/uusdc", "60000000000")
	results := &coretypes.ResultBlockResults{
		Height: *height,
		TxsResults: []*abci.ExecTxResult{
			{Events: []abci.Event{event("message", "action", "transfer"), e, e}},
			{Code: 1, Events: []abci.Event{e, e}},
		},
		FinalizeBlockEvents: []abci.Event{event("message"), event("coin_spent", "spender", "escrow", "amount", "100000000001ibc/USDC")},
	}
	if r.incomplete {
		results.TxsResults = results.TxsResults[:1]
	}
	return results, nil
}

func rpcTestHub(r *alertRPC) *cosmosrpc.Hub {
	l := zerolog.Nop()
	hub := cosmosrpc.NewHub(nil, &l, time.Second)
	hub.Clients = []cosmosrpc.CosmosJSONRPCClient{r}
	return hub
}

func rpcTestDB(t *testing.T) (*sql.DB, *gorm.DB) {
	t.Helper()
	db := alertDB(t)
	gdb, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: db}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return db, gdb
}

func rpcCheckpoint(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var height int64
	if err := db.QueryRow(`SELECT height FROM transfer_alert_rpc_cursors WHERE chain_id = 'test-1'`).Scan(&height); err != nil {
		t.Fatal(err)
	}
	return height
}

func TestDBRPCAtomicCheckpointAndIndexerOverlap(t *testing.T) {
	db, gdb := rpcTestDB(t)
	ctx := context.Background()
	r := &alertRPC{chain: "test-1", head: 42}
	hub := rpcTestHub(r)
	if err := hub.Rebalance(ctx); err != nil {
		t.Fatal(err)
	}
	if height, err := initRPCCursor(ctx, gdb, r.chain, 0, 42); err != nil || height != 41 {
		t.Fatalf("initial cursor: %d %v", height, err)
	}
	if _, err := db.Exec(`ALTER TABLE transfer_alert_rpc_cursors ADD CONSTRAINT fail_checkpoint CHECK (height <> 42)`); err != nil {
		t.Fatal(err)
	}
	if err := ingestRPCBlock(ctx, gdb, hub, r.chain, 42); err == nil {
		t.Fatal("checkpoint failure should roll back event groups")
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM transfer_alert_batches`).Scan(&count); err != nil || count != 0 || rpcCheckpoint(t, db) != 41 {
		t.Fatalf("partial commit: groups=%d err=%v", count, err)
	}
	if _, err := db.Exec(`ALTER TABLE transfer_alert_rpc_cursors DROP CONSTRAINT fail_checkpoint`); err != nil {
		t.Fatal(err)
	}
	if err := ingestRPCBlock(ctx, gdb, hub, r.chain, 42); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM transfer_alert_batches`).Scan(&count); err != nil || count != 2 || rpcCheckpoint(t, db) != 42 {
		t.Fatalf("expected successful transaction plus finalize group: %d %v", count, err)
	}
	c := testConfig(t)
	if n, err := scanBatch(ctx, db, c); err != nil || n != 2 {
		t.Fatalf("evaluate RPC events: %d %v", n, err)
	}
	// Feed the same block through the normal indexer's insertion path afterwards.
	height := int64(42)
	block, _ := r.Block(ctx, &height)
	results, _ := r.BlockResults(ctx, &height)
	data, err := mq.NewBlockResultMsgBytes(block, results)
	if err != nil {
		t.Fatal(err)
	}
	var message mq.BlockResultMsg
	if err := json.Unmarshal(data, &message); err != nil {
		t.Fatal(err)
	}
	if err := coredb.InsertTransferAlertBatches(ctx, gdb, c.Chain, &message); err != nil {
		t.Fatal(err)
	}
	if n, err := scanBatch(ctx, db, c); err != nil || n != 0 {
		t.Fatalf("backfill reset evaluated groups: %d %v", n, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM transfer_alerts`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("duplicate alerts: %d %v", count, err)
	}
	if height, err := initRPCCursor(ctx, gdb, c.Chain, 10, 100); err != nil || height != 42 {
		t.Fatalf("restart ignored persisted cursor: %d %v", height, err)
	}
	// The reverse order must also be harmless: backfill first, then the RPC reader.
	height = 43
	block, _ = r.Block(ctx, &height)
	results, _ = r.BlockResults(ctx, &height)
	data, err = mq.NewBlockResultMsgBytes(block, results)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &message); err != nil {
		t.Fatal(err)
	}
	if err := coredb.InsertTransferAlertBatches(ctx, gdb, c.Chain, &message); err != nil {
		t.Fatal(err)
	}
	if _, err := scanBatch(ctx, db, c); err != nil {
		t.Fatal(err)
	}
	if err := ingestRPCBlock(ctx, gdb, hub, c.Chain, 43); err != nil {
		t.Fatal(err)
	}
	if n, err := scanBatch(ctx, db, c); err != nil || n != 0 || rpcCheckpoint(t, db) != 43 {
		t.Fatalf("RPC replay reset evaluated groups: %d %v", n, err)
	}
}

func TestDBRPCRejectsWrongChainAndIncompleteResults(t *testing.T) {
	db, gdb := rpcTestDB(t)
	ctx := context.Background()
	if _, err := initRPCCursor(ctx, gdb, "test-1", 42, 42); err != nil {
		t.Fatal(err)
	}
	for _, r := range []*alertRPC{{chain: "another-1", head: 42}, {chain: "test-1", head: 42, incomplete: true}} {
		hub := rpcTestHub(r)
		if err := hub.Rebalance(ctx); err != nil {
			t.Fatal(err)
		}
		if err := ingestRPCBlock(ctx, gdb, hub, "test-1", 42); err == nil {
			t.Fatal("invalid RPC block accepted")
		}
		if rpcCheckpoint(t, db) != 41 {
			t.Fatal("checkpoint advanced past an invalid response")
		}
	}
}

func TestDBRPCRetryAndRestartDoNotSkipHeights(t *testing.T) {
	db, gdb := rpcTestDB(t)
	r := &alertRPC{chain: "test-1", head: 42, failNext: true}
	c := RPCConfig{Chain: "test-1"}
	runUntil := func(target int64) {
		t.Helper()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- runRPC(ctx, c, gdb, rpcTestHub(r)) }()
		deadline := time.Now().Add(10 * time.Second)
		for {
			var height int64
			err := db.QueryRow(`SELECT height FROM transfer_alert_rpc_cursors WHERE chain_id = 'test-1'`).Scan(&height)
			if err == nil && height == target {
				break
			}
			if time.Now().After(deadline) {
				cancel()
				<-done
				t.Fatalf("cursor did not reach %d: height=%d err=%v", target, height, err)
			}
			time.Sleep(10 * time.Millisecond)
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("reader did not stop cleanly: %v", err)
		}
	}
	runUntil(44)
	r.head = 46
	runUntil(46)
	want := []int64{42, 42, 43, 44, 45, 46}
	if fmt.Sprint(r.calls) != fmt.Sprint(want) {
		t.Fatalf("requested heights: got %v want %v", r.calls, want)
	}
}
