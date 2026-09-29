package repositories

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/core-indexer/api/dto"
)

func TestGetValidatorsSearchBinding(t *testing.T) {
	// Every nonempty search must use these same templates, independent of its
	// contents. WithArgs checks literal values, including existing LIKE wildcards.
	const selectSQL = `SELECT validators.*, validator_vote_counts.last_10000 AS last_10000 FROM "validators" LEFT JOIN validator_vote_counts ON validators.operator_address = validator_vote_counts.validator_address WHERE is_active = $1 AND (moniker ILIKE $2 OR operator_address = $3) ORDER BY "voting_power" DESC,"moniker" LIMIT $4`
	const countSQL = `SELECT count(*) FROM "validators" WHERE is_active = $1 AND (moniker ILIKE $2 OR operator_address = $3)`
	searches := []struct{ name, value string }{
		{"ordinary", "validator"},
		{"quote predicate", "probe' OR '1'='1"},
		{"statement and comment", "probe'); SELECT 1; --"},
		{"unicode and backslash", "ผู้ตรวจสอบ\\name'"},
		{"existing wildcards", "probe_%"},
	}
	for _, search := range searches {
		for _, countTotal := range []bool{false, true} {
			mode := "without count"
			if countTotal {
				mode = "with count"
			}
			t.Run(search.name+"/"+mode, func(t *testing.T) {
				db, mock := newSQLMockDB(t)
				mock.ExpectQuery(selectSQL).
					WithArgs(true, "%"+search.value+"%", search.value, 10).
					WillReturnRows(sqlmock.NewRows([]string{"operator_address", "moniker"}).AddRow("operator", search.value))
				if countTotal {
					mock.ExpectBegin()
					mock.ExpectExec("SET LOCAL statement_timeout = '250ms'").WithArgs().WillReturnResult(sqlmock.NewResult(0, 0))
					mock.ExpectQuery(countSQL).
						WithArgs(true, "%"+search.value+"%", search.value).
						WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
					mock.ExpectExec("RESET statement_timeout").WithArgs().WillReturnResult(sqlmock.NewResult(0, 0))
					mock.ExpectCommit()
				}
				repo := NewValidatorRepository(db, 250*time.Millisecond)
				records, total, err := repo.GetValidators(dto.PaginationQuery{
					Limit: 10, Reverse: true, CountTotal: countTotal,
				}, dto.ValidatorStatusFilterActive, "", search.value)
				require.NoError(t, err)
				require.Len(t, records, 1)
				require.Equal(t, search.value, records[0].Moniker)
				if countTotal {
					require.EqualValues(t, 1, total)
				} else {
					require.Zero(t, total)
				}
			})
		}
	}
}
