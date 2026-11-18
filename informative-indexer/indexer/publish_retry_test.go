package indexer

import (
	"bytes"
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"cloud.google.com/go/pubsub/v2/pstest"
	"github.com/DATA-DOG/go-sqlmock"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	initiaapp "github.com/initia-labs/initia/app"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/cacher"
	indexererrors "github.com/initia-labs/core-indexer/pkg/errors"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

type failFirstPublish struct {
	calls atomic.Int32
	fail  bool
}

func (r *failFirstPublish) React(interface{}) (bool, interface{}, error) {
	if r.calls.Add(1) == 1 && r.fail {
		return true, nil, status.Error(codes.PermissionDenied, "temporary publish permission failure")
	}
	return false, nil, nil
}

type failFirstUpload struct{ calls int }

func (s *failFirstUpload) UploadFile(string, string, []byte) error {
	s.calls++
	if s.calls == 1 {
		return errors.New("claim-check upload unavailable")
	}
	return nil
}
func (s *failFirstUpload) ReadFile(string, string) ([]byte, error) { panic("unexpected read") }

func TestTxPublishFailureRetriesBlock(t *testing.T) {
	for _, failure := range []string{"publish", "claim-check", "invalid-transaction"} {
		t.Run(failure, func(t *testing.T) {
			log := zerolog.Nop()
			oldLogger := logger
			logger = &log
			t.Cleanup(func() { logger = oldLogger })
			reactor := &failFirstPublish{fail: failure == "publish"}
			server := pstest.NewServer(pstest.ServerReactorOption{FuncName: "Publish", Reactor: reactor})
			defer server.Close()
			t.Setenv("PUBSUB_EMULATOR_HOST", server.Addr)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			topic := "projects/test-project/topics/responses"
			if _, err := server.GServer.CreateTopic(ctx, &pubsubpb.Topic{Name: topic}); err != nil {
				t.Fatal(err)
			}
			producer, err := mq.NewProducer(ctx, "test-project", "")
			if err != nil {
				t.Fatal(err)
			}
			defer producer.Close()
			conn, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			database, err := gorm.Open(postgres.New(postgres.Config{Conn: conn}), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			encoding := initiaapp.MakeEncodingConfig()
			sender := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
			builder := encoding.TxConfig.NewTxBuilder()
			if err := builder.SetMsgs(&banktypes.MsgSend{FromAddress: sender, ToAddress: sender, Amount: sdk.NewCoins(sdk.NewInt64Coin("uinit", 1))}); err != nil {
				t.Fatal(err)
			}
			tx, err := encoding.TxConfig.TxEncoder()(builder.GetTx())
			if err != nil {
				t.Fatal(err)
			}
			block := mq.BlockResultMsg{Height: 7, Hash: "abcd", Timestamp: time.Now(), Txs: []mq.TxResult{{Hash: "abcd", Tx: tx, ExecTxResults: &abci.ExecTxResult{Events: []abci.Event{{Type: "message", Attributes: []abci.EventAttribute{{Key: "sender", Value: sender}}}}}}}}
			indexer := &Indexer{cacher: cacher.NewCacher(), dbClient: database, producer: producer, encodingConfig: &encoding, config: &Config{TxResponseTopic: "responses", ClaimCheckThresholdInMB: 1}}
			upload := &failFirstUpload{}
			if failure == "claim-check" {
				indexer.storageClient = upload
				indexer.config.ClaimCheckThresholdInMB = 0
			}
			attempts := 2
			if failure == "invalid-transaction" {
				block.Txs[0].ExecTxResults.Events = nil
				attempts = 1
			}
			for attempt := 0; attempt < attempts; attempt++ {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "blocks"`).WillReturnRows(sqlmock.NewRows([]string{"height"}).AddRow(7))
				if attempt == 0 {
					mock.ExpectRollback()
				} else {
					mock.ExpectQuery(`SELECT .* FROM "tracking"`).WillReturnRows(sqlmock.NewRows([]string{"latest_informative_block_height", "tx_count"}).AddRow(6, 0))
					mock.ExpectExec(`UPDATE "tracking" SET "tx_count"`).WillReturnResult(sqlmock.NewResult(0, 1))
					mock.ExpectExec(`UPDATE "tracking" SET "latest_informative_block_height"`).WillReturnResult(sqlmock.NewResult(0, 1))
					mock.ExpectExec(`INSERT INTO "transactions"`).WillReturnResult(sqlmock.NewResult(0, 1))
					mock.ExpectCommit()
				}
			}
			err = indexer.processUntilSucceeds(ctx, block)
			if failure == "invalid-transaction" {
				if !errors.Is(err, indexererrors.ErrorNonRetryable) {
					t.Fatalf("permanent parse error must not retry: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
			wantPublishes, wantMessages := int32(2), 1
			if failure == "claim-check" {
				wantPublishes = 1
				if upload.calls != 2 {
					t.Fatalf("upload attempts = %d, want 2", upload.calls)
				}
			}
			if failure == "invalid-transaction" {
				wantPublishes, wantMessages = 0, 0
			}
			if reactor.calls.Load() != wantPublishes {
				t.Fatalf("publish attempts = %d, want %d", reactor.calls.Load(), wantPublishes)
			}
			if len(server.Messages()) != wantMessages {
				t.Fatalf("published %d messages, want %d", len(server.Messages()), wantMessages)
			}
		})
	}
}
