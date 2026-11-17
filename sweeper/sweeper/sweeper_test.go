package sweeper

import (
	"context"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/initia-labs/core-indexer/pkg/mq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"testing"
	"time"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/initia-labs/core-indexer/pkg/cosmosrpc"
	"github.com/rs/zerolog"
)

type retryHub struct {
	cosmosrpc.CosmosJSONRPCHub
	heights chan int64
	block   func(context.Context) (*coretypes.ResultBlock, error)
	results func(context.Context) (*coretypes.ResultBlockResults, error)
}

func (h *retryHub) Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error) {
	if h.heights != nil {
		h.heights <- *height
	}
	return h.block(ctx)
}
func (h *retryHub) BlockResults(ctx context.Context, _ *int64) (*coretypes.ResultBlockResults, error) {
	return h.results(ctx)
}

func TestRPCRetriesObserveCancellation(t *testing.T) {
	log := zerolog.Nop()
	logger = &log
	for _, method := range []string{"block", "results"} {
		for _, inFlight := range []bool{false, true} {
			name := method + "/retry sleep"
			if inFlight {
				name = method + "/in flight"
			}
			t.Run(name, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				entered := make(chan struct{})
				call := func(ctx context.Context) error {
					close(entered)
					if inFlight {
						<-ctx.Done()
						return ctx.Err()
					}
					return errors.New("RPC unavailable")
				}
				hub := &retryHub{
					block:   func(ctx context.Context) (*coretypes.ResultBlock, error) { return nil, call(ctx) },
					results: func(ctx context.Context) (*coretypes.ResultBlockResults, error) { return nil, call(ctx) },
				}
				s := &Sweeper{rpcClient: hub}
				done := make(chan error, 1)
				go func() {
					var err error
					if method == "block" {
						_, err = s.GetBlock(ctx, 7)
					} else {
						_, err = s.GetBlockResults(ctx, 7)
					}
					done <- err
				}()
				select {
				case <-entered:
				case <-time.After(2 * time.Second):
					t.Fatal("RPC not called")
				}
				cancel()
				select {
				case err := <-done:
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("error = %v", err)
					}
				case <-time.After(500 * time.Millisecond):
					t.Fatal("retry did not stop on cancellation")
				}
			})
		}
	}
}

func TestStartSweepingRetriesSameHeightAndCancels(t *testing.T) {
	log := zerolog.Nop()
	logger = &log
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	database, err := gorm.Open(postgres.New(postgres.Config{Conn: conn}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT .* FROM "tracking"`).WillReturnRows(sqlmock.NewRows([]string{"latest_informative_block_height"}).AddRow(6))
	producer, err := mq.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "", "log_level": 0})
	if err != nil {
		t.Fatal(err)
	}
	defer producer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	heights := make(chan int64, 3)
	calls := 0
	hub := &retryHub{
		heights: heights,
		// Incomplete block reaches serialization and returns an error.
		block: func(ctx context.Context) (*coretypes.ResultBlock, error) {
			calls++
			if calls == 2 {
				<-ctx.Done()
				return nil, ctx.Err()
			}
			return &coretypes.ResultBlock{}, nil
		},
		results: func(context.Context) (*coretypes.ResultBlockResults, error) {
			return &coretypes.ResultBlockResults{}, nil
		},
	}
	s := &Sweeper{rpcClient: hub, dbClient: database, producer: producer, config: &SweeperConfig{}}
	done := make(chan struct{})
	go func() { defer close(done); s.StartSweeping(ctx) }()
	for range 2 {
		select {
		case height := <-heights:
			if height != 7 {
				t.Errorf("retried height %d, want 7", height)
			}
		case <-done:
			t.Fatal("sweeper stopped instead of retrying")
		case <-time.After(3 * time.Second):
			t.Fatal("sweeper did not retry")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("sweeper did not stop")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
