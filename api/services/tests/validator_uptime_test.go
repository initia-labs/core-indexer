package services_test

import (
	"encoding/json"
	"math"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/routes"
	"github.com/initia-labs/core-indexer/pkg/db"
)

type uptimeBlockRepository struct {
	repositories.BlockRepositoryI
	height int64
}

func (r *uptimeBlockRepository) GetLatestBlock() (*db.Block, error) {
	return &db.Block{Height: r.height, Timestamp: time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)}, nil
}

type uptimeValidatorRepository struct {
	repositories.ValidatorRepositoryI
	proposedRange  [2]int64
	signatureRange [2]int64
}

func (r *uptimeValidatorRepository) GetValidatorBlockVoteByBlockLimit(minHeight, maxHeight int64) ([]dto.ValidatorBlockVoteModel, error) {
	r.proposedRange = [2]int64{minHeight, maxHeight}
	return []dto.ValidatorBlockVoteModel{{Height: maxHeight, Vote: "PROPOSE"}}, nil
}

func (r *uptimeValidatorRepository) GetValidatorCommitSignatures(_ string, minHeight, maxHeight int64) ([]dto.ValidatorBlockVoteModel, error) {
	r.signatureRange = [2]int64{minHeight, maxHeight}
	votes := []dto.ValidatorBlockVoteModel{{Height: maxHeight, Vote: "PROPOSE"}}
	if minHeight < maxHeight {
		votes = append(votes, dto.ValidatorBlockVoteModel{Height: maxHeight - 1, Vote: "ABSTAIN"})
	}
	return votes, nil
}

func (r *uptimeValidatorRepository) GetValidatorSlashEvents(string, time.Time) ([]dto.ValidatorUptimeEventModel, error) {
	return []dto.ValidatorUptimeEventModel{}, nil
}

func (r *uptimeValidatorRepository) GetValidatorUptimeInfo(string) (*dto.ValidatorWithVoteCountModel, error) {
	return &dto.ValidatorWithVoteCountModel{
		Validator:          db.Validator{IsActive: true},
		ValidatorVoteCount: db.ValidatorVoteCount{Last10000: 10000},
	}, nil
}

func TestValidatorUptimeWorkWindow(t *testing.T) {
	for _, tc := range []struct {
		name      string
		blocks    int
		height    int64
		wantTotal int
		wantMin   int64
	}{
		{"one block", 1, 25000, 1, 25000},
		{"recent blocks", 100, 25000, 100, 24901},
		{"below cap", 9999, 25000, 9999, 15002},
		{"at cap", 10000, 25000, 10000, 15001},
		{"above cap", 10001, 25000, 10000, 15001},
		{"maximum integer", math.MaxInt, 25000, 10000, 15001},
		{"young chain", 10000, 50, 50, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			validatorRepo := &uptimeValidatorRepository{}
			app := fiber.New()
			routes.SetupValidatorRoutes(app, validatorRepo, &uptimeBlockRepository{height: tc.height}, nil)
			request := httptest.NewRequest("GET", "/indexer/validator/v1/validators/test-validator/uptime?blocks="+strconv.Itoa(tc.blocks), nil)
			response, err := app.Test(request)
			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, fiber.StatusOK, response.StatusCode)

			// Both repository queries must receive the bounded interval, not merely
			// have their results truncated when building the HTTP response.
			wantRange := [2]int64{tc.wantMin, tc.height}
			require.Equal(t, wantRange, validatorRepo.proposedRange)
			require.Equal(t, wantRange, validatorRepo.signatureRange)

			var body dto.ValidatorUptimeResponse
			require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
			require.Equal(t, dto.ValidatorUptimeSummary{
				Total:          tc.wantTotal,
				ProposedBlocks: 1,
				MissedBlocks:   min(1, tc.wantTotal-1),
				SignedBlocks:   max(0, tc.wantTotal-2),
			}, body.Uptime)
			require.Len(t, body.Recent100Blocks, min(100, tc.wantTotal))
			require.Equal(t, dto.ValidatorBlockVoteModel{Height: tc.height, Vote: "PROPOSE"}, body.Recent100Blocks[0])
			for i, block := range body.Recent100Blocks {
				require.Equal(t, tc.height-int64(i), block.Height)
			}
		})
	}
}
