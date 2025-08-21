package dto

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestPaginationFromQuery(t *testing.T) {

	// Table-driven tests for various scenarios
	testCases := []struct {
		name           string
		queryString    string
		expectError    bool
		expectedError  *apperror.Response
		expectedResult *PaginationQuery
	}{
		{
			name:        "valid custom values",
			queryString: "pagination.limit=50&pagination.offset=100&pagination.key=dGVzdA==&pagination.reverse=false&pagination.count_total=true",
			expectError: false,
			expectedResult: &PaginationQuery{
				Limit:      50,
				Offset:     100,
				Key:        "dGVzdA==",
				Reverse:    false,
				CountTotal: true,
			},
		},
		{
			name:        "partial parameters",
			queryString: "pagination.limit=25&pagination.offset=50",
			expectError: false,
			expectedResult: &PaginationQuery{
				Limit:      25,
				Offset:     50,
				Key:        "",
				Reverse:    true,
				CountTotal: false,
			},
		},
		{
			name:          "invalid limit - not integer",
			queryString:   "pagination.limit=abc",
			expectError:   true,
			expectedError: apperror.NewLimitInteger(),
		},
		{
			name:          "invalid limit - too small",
			queryString:   "pagination.limit=0",
			expectError:   true,
			expectedError: apperror.NewInvalidLimit(),
		},
		{
			name:          "invalid limit - too large",
			queryString:   "pagination.limit=1001",
			expectError:   true,
			expectedError: apperror.NewInvalidLimit(),
		},
		{
			name:          "invalid offset - not integer",
			queryString:   "pagination.offset=xyz",
			expectError:   true,
			expectedError: apperror.NewOffsetInteger(),
		},
		{
			name:          "invalid offset - negative",
			queryString:   "pagination.offset=-10",
			expectError:   true,
			expectedError: apperror.NewOffsetInteger(),
		},
		{
			name:          "invalid reverse - not boolean",
			queryString:   "pagination.reverse=maybe",
			expectError:   true,
			expectedError: apperror.NewReverse(),
		},
		{
			name:          "invalid count_total - not boolean",
			queryString:   "pagination.count_total=perhaps",
			expectError:   true,
			expectedError: apperror.NewCountTotal(),
		},
		{
			name:        "valid boolean values",
			queryString: "pagination.reverse=true&pagination.count_total=false",
			expectError: false,
			expectedResult: &PaginationQuery{
				Limit:      10,
				Offset:     0,
				Key:        "",
				Reverse:    true,
				CountTotal: false,
			},
		},
		{
			name:        "edge case - limit at minimum",
			queryString: "pagination.limit=1",
			expectError: false,
			expectedResult: &PaginationQuery{
				Limit:      1,
				Offset:     0,
				Key:        "",
				Reverse:    true,
				CountTotal: false,
			},
		},
		{
			name:        "edge case - limit at maximum",
			queryString: "pagination.limit=1000",
			expectError: false,
			expectedResult: &PaginationQuery{
				Limit:      1000,
				Offset:     0,
				Key:        "",
				Reverse:    true,
				CountTotal: false,
			},
		},
		{
			name:        "edge case - offset at minimum",
			queryString: "pagination.offset=0",
			expectError: false,
			expectedResult: &PaginationQuery{
				Limit:      10,
				Offset:     0,
				Key:        "",
				Reverse:    true,
				CountTotal: false,
			},
		},
		{
			name:        "large offset",
			queryString: "pagination.offset=999999",
			expectError: false,
			expectedResult: &PaginationQuery{
				Limit:      10,
				Offset:     999999,
				Key:        "",
				Reverse:    true,
				CountTotal: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			req := &fasthttp.RequestCtx{}
			req.URI().SetQueryString(tc.queryString)
			c := app.AcquireCtx(req)
			defer app.ReleaseCtx(c)

			result, err := PaginationFromQuery(c)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)

				// Assert specific error type and message
				if tc.expectedError != nil {
					var appErr *apperror.Response
					assert.ErrorAs(t, err, &appErr)
					assert.Equal(t, tc.expectedError.Code, appErr.Code)
					assert.Equal(t, tc.expectedError.Message, appErr.Message)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.expectedResult.Limit, result.Limit)
				assert.Equal(t, tc.expectedResult.Offset, result.Offset)
				assert.Equal(t, tc.expectedResult.Key, result.Key)
				assert.Equal(t, tc.expectedResult.Reverse, result.Reverse)
				assert.Equal(t, tc.expectedResult.CountTotal, result.CountTotal)
			}
		})
	}
}
