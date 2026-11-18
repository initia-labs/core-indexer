package indexer

import (
	"bytes"
	"context"
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

	"github.com/initia-labs/core-indexer/pkg/mq"
)

type failFirstPublish struct{ calls atomic.Int32 }

func (r *failFirstPublish) React(interface{}) (bool, interface{}, error) {
	if r.calls.Add(1) == 1 {
		return true, nil, status.Error(codes.PermissionDenied, "temporary publish permission failure")
	}
	return false, nil, nil
}

func TestTxPublishFailureRetriesBlock(t *testing.T) {
	log := zerolog.Nop()
	oldLogger := logger
	logger = &log
	t.Cleanup(func() { logger = oldLogger })
	reactor := &failFirstPublish{}
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
	indexer := &Indexer{dbClient: database, producer: producer, encodingConfig: &encoding, config: &IndexerConfig{TxResponseTopic: "responses", ClaimCheckThresholdInMB: 1}}
	for attempt := 0; attempt < 2; attempt++ {
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT .* FROM "validators"`).WillReturnRows(sqlmock.NewRows([]string{"operator_address", "account_id", "consensus_address"}))
		mock.ExpectQuery(`INSERT INTO "blocks"`).WillReturnRows(sqlmock.NewRows([]string{"height"}).AddRow(7))
		if attempt == 0 {
			mock.ExpectRollback()
		} else {
			mock.ExpectExec(`INSERT INTO "transactions"`).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
		}
	}
	if err := indexer.processUntilSucceeds(ctx, block); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	if reactor.calls.Load() != 2 {
		t.Fatalf("publish attempts = %d, want 2", reactor.calls.Load())
	}
	for _, message := range server.Messages() {
		if message.Topic != topic {
			t.Fatalf("unexpected DLQ message: %s", message.Topic)
		}
	}
	if len(server.Messages()) != 1 {
		t.Fatal("response was not published")
	}
}
