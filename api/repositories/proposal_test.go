package repositories

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestProposalVotesOptionalCount(t *testing.T) {
	const joins = ` FROM "proposal_votes" LEFT JOIN transactions ON proposal_votes.transaction_id = transactions.id LEFT JOIN blocks ON transactions.block_height = blocks.height LEFT JOIN validators ON proposal_votes.validator_address = validators.operator_address WHERE proposal_votes.proposal_id = $1`
	const columns = `proposal_votes.proposal_id, proposal_votes.yes, proposal_votes.no, proposal_votes.no_with_veto, proposal_votes.abstain, proposal_votes.is_vote_weighted, proposal_votes.voter, transactions.hash as tx_hash, blocks.timestamp, validators.operator_address as validator_address, validators.moniker as validator_moniker, validators.identity as validator_identity`
	for _, countTotal := range []bool{false, true} {
		name := "without count"
		if countTotal {
			name = "with count"
		}
		t.Run(name, func(t *testing.T) {
			conn, mock := newSQLMockDB(t)
			if countTotal {
				mock.ExpectBegin()
				mock.ExpectExec("SET LOCAL statement_timeout = '250ms'").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery("SELECT count(*)" + joins).WithArgs(7).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
				mock.ExpectExec("RESET statement_timeout").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			}
			mock.ExpectQuery("SELECT "+columns+joins+` ORDER BY "transactions"."block_height" DESC,"transactions"."block_index" DESC LIMIT $2 OFFSET $3`).WithArgs(7, 1, 2).WillReturnRows(sqlmock.NewRows([]string{"voter"}).AddRow("voter"))
			votes, total, err := NewProposalRepository(conn, 250*time.Millisecond).GetProposalVotes(7, 1, 2, countTotal, "", "")
			require.NoError(t, err)
			require.Len(t, votes, 1)
			if countTotal {
				require.EqualValues(t, 3, total)
			} else {
				require.Zero(t, total)
			}
		})
	}
}
