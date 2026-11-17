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
	"github.com/initia-labs/core-indexer/api/services"
)

type paginatedTxService struct {
	services.TxService
	pagination *dto.PaginationQuery
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
