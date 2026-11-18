package repositories

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/core-indexer/api/dto"
)

const proposalTypesSQL = `SELECT DISTINCT json_array_elements_text(types) AS type FROM proposals`

func TestProposalTypesCatalog(t *testing.T) {
	for _, tc := range []struct {
		name string
		rows []string
		want dto.ProposalsTypesResponse
	}{
		{
			name: "legacy types precede message URLs",
			rows: []string{"/initia.move.v1.MsgExecute", "Text", "/cosmos.bank.v1beta1.MsgSend", "CommunityPoolSpend"},
			want: dto.ProposalsTypesResponse{"CommunityPoolSpend", "Text", "/cosmos.bank.v1beta1.MsgSend", "/initia.move.v1.MsgExecute"},
		},
		{name: "empty catalog", want: dto.ProposalsTypesResponse{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn, mock := newSQLMockDB(t)
			rows := sqlmock.NewRows([]string{"type"})
			for _, typ := range tc.rows {
				rows.AddRow(typ)
			}
			mock.ExpectQuery(proposalTypesSQL).WillReturnRows(rows)
			got, err := NewProposalRepository(conn, time.Second).GetAllProposalTypes()
			require.NoError(t, err)
			require.Equal(t, tc.want, *got)
			if len(tc.want) == 0 {
				encoded, err := json.Marshal(got)
				require.NoError(t, err)
				require.JSONEq(t, "[]", string(encoded))
			}
		})
	}
}

func TestProposalTypesQueryFailure(t *testing.T) {
	conn, mock := newSQLMockDB(t)
	queryErr := errors.New("database unavailable")
	mock.ExpectQuery(proposalTypesSQL).WillReturnError(queryErr)
	got, err := NewProposalRepository(conn, time.Second).GetAllProposalTypes()
	require.ErrorIs(t, err, queryErr)
	require.Nil(t, got)
}

func TestProposalTypesQueryTimeout(t *testing.T) {
	conn, mock := newSQLMockDB(t)
	mock.ExpectQuery(proposalTypesSQL).
		WillDelayFor(200 * time.Millisecond).
		WillReturnRows(sqlmock.NewRows([]string{"type"}).AddRow("Text"))
	got, err := NewProposalRepository(conn, 20*time.Millisecond).GetAllProposalTypes()
	require.ErrorIs(t, err, sqlmock.ErrCancelled)
	require.Nil(t, got, "a timed-out query must not return an incomplete catalog")
}
