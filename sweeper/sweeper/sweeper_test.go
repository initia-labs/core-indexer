package sweeper

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
)

type sweepClient struct {
	cosmosrpc.CosmosJSONRPCClient
	statusHeight int64
	block        func(context.Context, *int64) (*coretypes.ResultBlock, error)
	results      func(context.Context, *int64) (*coretypes.ResultBlockResults, error)
}

func (c *sweepClient) Status(context.Context) (*coretypes.ResultStatus, error) {
	return &coretypes.ResultStatus{SyncInfo: coretypes.SyncInfo{LatestBlockHeight: c.statusHeight}}, nil
}
func (c *sweepClient) GetIdentifier() string { return "test" }
func (c *sweepClient) Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error) {
	if c.block != nil {
		return c.block(ctx, height)
	}
	return &coretypes.ResultBlock{Block: &types.Block{Header: types.Header{Height: *height}}}, nil
}
func (c *sweepClient) BlockResults(ctx context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
	if c.results != nil {
		return c.results(ctx, height)
	}
	return &coretypes.ResultBlockResults{Height: *height}, nil
}

// Run the real hub and sweep loop with only the database and RPC endpoints faked.
func startTestSweep(t *testing.T, clients ...cosmosrpc.CosmosJSONRPCClient) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	log := zerolog.Nop()
	oldLogger := logger
	logger = &log
	t.Cleanup(func() { logger = oldLogger })
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	database, err := gorm.Open(postgres.New(postgres.Config{Conn: conn}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT .* FROM "tracking"`).WillReturnRows(sqlmock.NewRows([]string{"latest_informative_block_height"}).AddRow(6))
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	hub := cosmosrpc.NewHub(nil, &log, time.Second)
	hub.Clients = clients
	if err := hub.Rebalance(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Sweeper{rpcClient: hub, dbClient: database, config: &SweeperConfig{RebalanceInterval: 100}}
	done := make(chan struct{})
	go func() { defer close(done); s.StartSweeping(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("sweeper did not stop")
		}
	})
	return cancel, done
}

func TestSweepRPCCancellation(t *testing.T) {
	for _, method := range []string{"block", "results"} {
		for _, inFlight := range []bool{false, true} {
			name := method + "/retry wait"
			if inFlight {
				name = method + "/in flight"
			}
			t.Run(name, func(t *testing.T) {
				entered := make(chan struct{}, 1)
				call := func(ctx context.Context) error {
					select {
					case entered <- struct{}{}:
					default:
					}
					if inFlight {
						<-ctx.Done()
						return ctx.Err()
					}
					return errors.New("RPC unavailable")
				}
				client := &sweepClient{statusHeight: 10}
				if method == "block" {
					client.block = func(ctx context.Context, _ *int64) (*coretypes.ResultBlock, error) { return nil, call(ctx) }
				} else {
					client.results = func(ctx context.Context, _ *int64) (*coretypes.ResultBlockResults, error) { return nil, call(ctx) }
				}
				cancel, done := startTestSweep(t, client)
				select {
				case <-entered:
				case <-time.After(2 * time.Second):
					t.Fatal("RPC not called")
				}
				cancel()
				select {
				case <-done:
				case <-time.After(500 * time.Millisecond):
					t.Fatal("sweep did not stop on cancellation")
				}
			})
		}
	}
}

func TestSweepFailsOverRejectedPair(t *testing.T) {
	advanced := make(chan int64, 1)
	bad := &sweepClient{
		statusHeight: 20, // Make the malformed endpoint the first choice.
		block: func(ctx context.Context, height *int64) (*coretypes.ResultBlock, error) {
			if *height != 7 {
				advanced <- *height
				<-ctx.Done()
				return nil, ctx.Err()
			}
			return &coretypes.ResultBlock{Block: &types.Block{Header: types.Header{Height: *height}}}, nil
		},
		results: func(_ context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
			return &coretypes.ResultBlockResults{Height: *height, TxsResults: []*abci.ExecTxResult{{}}}, nil
		},
	}
	var goodBlockCalls, goodResultsCalls int
	good := &sweepClient{
		statusHeight: 10,
		block: func(_ context.Context, height *int64) (*coretypes.ResultBlock, error) {
			goodBlockCalls++
			return &coretypes.ResultBlock{Block: &types.Block{Header: types.Header{Height: *height}, Data: types.Data{Txs: types.Txs{types.Tx("tx")}}}}, nil
		},
		results: func(_ context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
			goodResultsCalls++
			return &coretypes.ResultBlockResults{Height: *height, TxsResults: []*abci.ExecTxResult{{}}}, nil
		},
	}
	cancel, done := startTestSweep(t, bad, good)
	select {
	case height := <-advanced:
		if height != 8 {
			t.Fatalf("advanced to height %d, want 8", height)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("malformed endpoint prevented failover and progress")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("sweep did not stop")
	}
	if goodBlockCalls != 1 || goodResultsCalls != 1 {
		t.Fatalf("healthy endpoint pair not used: block=%d results=%d", goodBlockCalls, goodResultsCalls)
	}
}

func TestSweepRetriesSameHeightWhenAllPairsFail(t *testing.T) {
	heights := make(chan int64, 4)
	client := &sweepClient{
		statusHeight: 10,
		block: func(_ context.Context, height *int64) (*coretypes.ResultBlock, error) {
			heights <- *height
			return &coretypes.ResultBlock{Block: &types.Block{Header: types.Header{Height: *height}}}, nil
		},
		results: func(_ context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
			return &coretypes.ResultBlockResults{Height: *height, TxsResults: []*abci.ExecTxResult{{}}}, nil
		},
	}
	cancel, done := startTestSweep(t, client)
	// The fourth call is in the next sweep attempt, after handleQuery's retries.
	for range cosmosrpc.MAX_RETRY_COUNT + 1 {
		select {
		case height := <-heights:
			if height != 7 {
				t.Fatalf("skipped failed height: got %d", height)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("sweep did not retry")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("sweep did not stop")
	}
}
