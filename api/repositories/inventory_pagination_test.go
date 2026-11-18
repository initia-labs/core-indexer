package repositories

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/core-indexer/api/dto"
)

func TestHistoricalPowerPageUsesSQLLimitAndFullCount(t *testing.T) {
	conn, mock := newSQLMockDB(t)
	repo := NewValidatorRepository(conn, 250*time.Millisecond)
	const pageSQL = `SELECT hour_rounded_timestamp, timestamp, voting_power FROM "validator_historical_powers" WHERE validator_address = $1 AND timestamp >= $2 ORDER BY "timestamp" DESC,"hour_rounded_timestamp" DESC LIMIT $3 OFFSET $4`
	const countSQL = `SELECT count(*) FROM "validator_historical_powers" WHERE validator_address = $1 AND timestamp >= $2`
	pagination := &dto.PaginationQuery{Limit: 2, Offset: 4, Reverse: true, CountTotal: true}

	mock.ExpectQuery(pageSQL).WithArgs("operator", sqlmock.AnyArg(), 2, 4).
		WillReturnRows(sqlmock.NewRows([]string{"hour_rounded_timestamp", "timestamp", "voting_power"}))
	mock.ExpectBegin()
	mock.ExpectExec("SET LOCAL statement_timeout = '250ms'").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(countSQL).WithArgs("operator", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	mock.ExpectExec("RESET statement_timeout").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	rows, total, err := repo.GetValidatorHistoricalPowers(pagination, "operator")
	require.NoError(t, err)
	require.NotNil(t, rows)
	require.Empty(t, rows)
	require.EqualValues(t, 4, total)
}

func TestAccountCollectionPageUsesSQLLimitAndDistinctCount(t *testing.T) {
	conn, mock := newSQLMockDB(t)
	repo := NewNftRepository(conn, 250*time.Millisecond)
	const owned = ` FROM "collections" JOIN nfts ON nfts.collection = collections.id AND nfts.owner = $1 AND nfts.is_burned = false`
	const pageSQL = `SELECT collections.name, collections.creator, collections.uri, collections.description, collections.id, COUNT(nfts.id) AS count` +
		owned + ` GROUP BY "collections"."id" ORDER BY "collections"."name","collections"."id" LIMIT $2 OFFSET $3`
	const countSQL = `SELECT COUNT(DISTINCT("collections"."id"))` + owned

	for _, countTotal := range []bool{true, false} {
		pagination := &dto.PaginationQuery{Limit: 2, Offset: 4, CountTotal: countTotal}
		mock.ExpectQuery(pageSQL).WithArgs("0x1", 2, 4).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "count"}).AddRow("0x2", "collection", 7))
		expectedTotal := int64(-1)
		if countTotal {
			mock.ExpectBegin()
			mock.ExpectExec("SET LOCAL statement_timeout = '250ms'").WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery(countSQL).WithArgs("0x1").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
			mock.ExpectExec("RESET statement_timeout").WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()
			expectedTotal = 5
		}
		rows, total, err := repo.GetCollectionsByAccountAddress(pagination, "0x1")
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, "0x2", rows[0].ID)
		require.EqualValues(t, 7, rows[0].Count)
		require.Equal(t, expectedTotal, total)
	}
}
