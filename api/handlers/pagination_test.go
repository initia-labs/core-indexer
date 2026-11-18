package handlers

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/initia-labs/core-indexer/api/dto"
	"github.com/initia-labs/core-indexer/api/repositories"
	"github.com/initia-labs/core-indexer/api/services"
)

type paginatedTxService struct {
	services.TxService
	pagination *dto.PaginationQuery
}

type paginationInventoryRepository struct {
	repositories.NftRepositoryI
	repositories.ValidatorRepositoryI
	queries []*dto.PaginationQuery
}

func (r *paginationInventoryRepository) GetCollectionsByAccountAddress(pagination *dto.PaginationQuery, _ string) ([]dto.CollectionByAccountAddressModel, int64, error) {
	r.queries = append(r.queries, pagination)
	return []dto.CollectionByAccountAddressModel{}, 20, nil
}

func (r *paginationInventoryRepository) GetValidatorHistoricalPowers(pagination *dto.PaginationQuery, _ string) ([]dto.ValidatorHistoricalPowerModel, int64, error) {
	r.queries = append(r.queries, pagination)
	return []dto.ValidatorHistoricalPowerModel{}, 20, nil
}

func TestInventoryRoutesSupportOptionalPagination(t *testing.T) {
	for _, path := range []string{"/validators/operator/historical_powers", "/collections/by_account/0x1"} {
		t.Run(path, func(t *testing.T) {
			repo := &paginationInventoryRepository{}
			app := fiber.New()
			app.Get("/validators/:operatorAddr/historical_powers", NewValidatorHandler(services.NewValidatorService(repo, nil, nil)).GetValidatorHistoricalPowers)
			app.Get("/collections/by_account/:accountAddress", NewNftHandler(services.NewNftService(repo)).GetCollectionsByAccountAddress)

			for _, tc := range []struct {
				query      string
				status     int
				pagination *dto.PaginationQuery
				nextKey    string
			}{
				{status: fiber.StatusOK},
				{
					query:      "?pagination.limit=5&pagination.offset=5&pagination.reverse=false",
					status:     fiber.StatusOK,
					pagination: &dto.PaginationQuery{Limit: 5, Offset: 5, CountTotal: true},
					nextKey:    base64.StdEncoding.EncodeToString([]byte("10")),
				},
				{query: "?pagination.offset=10001", status: fiber.StatusBadRequest},
			} {
				previousCalls := len(repo.queries)
				response, err := app.Test(httptest.NewRequest("GET", path+tc.query, nil))
				require.NoError(t, err)
				require.Equal(t, tc.status, response.StatusCode)
				if tc.status == fiber.StatusBadRequest {
					response.Body.Close()
					require.Len(t, repo.queries, previousCalls)
					continue
				}
				var body struct {
					Pagination dto.PaginationResponse `json:"pagination"`
				}
				err = json.NewDecoder(response.Body).Decode(&body)
				response.Body.Close()
				require.NoError(t, err)
				require.Len(t, repo.queries, previousCalls+1)
				require.Equal(t, tc.pagination, repo.queries[previousCalls])
				require.Equal(t, "20", body.Pagination.Total)
				if tc.nextKey == "" {
					require.Nil(t, body.Pagination.NextKey)
				} else {
					require.NotNil(t, body.Pagination.NextKey)
					require.Equal(t, tc.nextKey, *body.Pagination.NextKey)
				}
			}
		})
	}
}

func (s *paginatedTxService) GetTxs(pagination *dto.PaginationQuery) (*dto.TxsModelResponse, error) {
	s.pagination = pagination
	return &dto.TxsModelResponse{}, nil
}

func TestPaginationOffsetLimitBeforeService(t *testing.T) {
	for _, offset := range []int{0, 10000, 10001, 999999, math.MaxInt} {
		for _, key := range []bool{false, true} {
			value := strconv.Itoa(offset)
			query := "pagination.offset=" + value
			if key {
				// A small explicit offset must not bypass the cursor's bound.
				query = "pagination.offset=0&pagination.key=" + url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(value)))
			}
			t.Run(query, func(t *testing.T) {
				service := &paginatedTxService{}
				app := fiber.New()
				app.Get("/txs", NewTxHandler(service).GetTxs)
				response, err := app.Test(httptest.NewRequest("GET", "/txs?"+query, nil))
				require.NoError(t, err)
				defer response.Body.Close()
				if offset <= 10000 {
					require.Equal(t, fiber.StatusOK, response.StatusCode)
					require.NotNil(t, service.pagination)
					require.Equal(t, offset, service.pagination.Offset)
				} else {
					require.Equal(t, fiber.StatusBadRequest, response.StatusCode)
					require.Nil(t, service.pagination, "rejected pagination reached the service")
					var body apperror.Response
					require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
					require.Equal(t, "pagination.offset must not exceed 10000", body.Message)
				}
			})
		}
	}
}
