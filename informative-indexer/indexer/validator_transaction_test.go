package indexer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cometbft/cometbft/types"
	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/initia-labs/core-indexer/informative-indexer/indexer/cacher"
	"github.com/initia-labs/core-indexer/pkg/db"
	indexererrors "github.com/initia-labs/core-indexer/pkg/errors"
	"github.com/initia-labs/core-indexer/pkg/mq"
)

func TestValidatorFailureRollsBackBlockAndHeight(t *testing.T) {
	log := zerolog.Nop()
	oldLogger := logger
	logger = &log
	t.Cleanup(func() { logger = oldLogger })
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	database, err := gorm.Open(postgres.New(postgres.Config{Conn: conn}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	indexer := &Indexer{dbClient: database, cacher: cacher.NewCacher()}
	block := &mq.BlockResultMsg{Hash: "abcd", Height: 7, Timestamp: time.Now(), LastCommit: &types.Commit{Height: 6}}
	proposer := &db.ValidatorAddress{OperatorAddress: "validator"}
	failure := errors.New("signature write unavailable")
	for _, fail := range []bool{true, false} {
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "blocks"`).WillReturnRows(sqlmock.NewRows([]string{"height"}).AddRow(7))
		mock.ExpectQuery(`SELECT .* FROM "tracking"`).WillReturnRows(sqlmock.NewRows([]string{"latest_informative_block_height", "tx_count"}).AddRow(6, 0))
		mock.ExpectExec(`UPDATE "tracking" SET "tx_count"`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE "tracking" SET "latest_informative_block_height"`).WillReturnResult(sqlmock.NewResult(0, 1))
		insert := mock.ExpectExec(`INSERT INTO "validator_commit_signatures"`)
		if fail {
			insert.WillReturnError(failure)
			mock.ExpectRollback()
		} else {
			insert.WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()
		}
		err := indexer.processBlockResults(context.Background(), block, proposer)
		if fail {
			if !errors.Is(err, failure) || errors.Is(err, indexererrors.ErrorNonRetryable) {
				t.Fatalf("validator failure must roll back and remain retryable: %v", err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	}
}
