package services_test

import (
	"encoding/base64"
	"testing"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories/mocks"
	"github.com/initia-labs/core-indexer/api/services"
	"github.com/stretchr/testify/require"
)

func TestProposalVotesCountTotal(t *testing.T) {
	for _, countTotal := range []bool{false, true} {
		name := "without count"
		if countTotal {
			name = "with count"
		}
		t.Run(name, func(t *testing.T) {
			repo := mocks.NewMockProposalRepository()
			total := int64(-1)
			expectedTotal := "-1"
			if countTotal {
				total = 25
				expectedTotal = "25"
			}
			votes := make([]dto.ProposalVote, 10)
			repo.On("GetProposalVotes", 7, int64(10), int64(2), countTotal, "voter", "yes").Return(votes, total, nil).Once()
			result, err := services.NewProposalService(repo).GetProposalVotes(dto.PaginationQuery{Limit: 10, Offset: 2, CountTotal: countTotal}, 7, "voter", "yes")
			require.NoError(t, err)
			require.Len(t, result.Votes, 10)
			require.Equal(t, expectedTotal, result.Pagination.Total)
			require.NotNil(t, result.Pagination.NextKey)
			next, err := base64.StdEncoding.DecodeString(*result.Pagination.NextKey)
			require.NoError(t, err)
			require.Equal(t, "12", string(next))
			repo.AssertExpectations(t)
		})
	}
}
