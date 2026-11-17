package services_test

import (
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories/mocks"
	"github.com/initia-labs/core-indexer/api/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProposalVotesCountTotal(t *testing.T) {
	for _, countTotal := range []bool{false, true} {
		name := "without count"
		if countTotal {
			name = "with count"
		}
		t.Run(name, func(t *testing.T) {
			repo := mocks.NewMockProposalRepository()
			repo.On("GetProposalVotes", 7, int64(10), int64(2), countTotal, "voter", "yes").Return([]dto.ProposalVote{}, int64(0), nil).Once()
			_, err := services.NewProposalService(repo).GetProposalVotes(dto.PaginationQuery{Limit: 10, Offset: 2, CountTotal: countTotal}, 7, "voter", "yes")
			require.NoError(t, err)
			repo.AssertExpectations(t)
		})
	}
}
