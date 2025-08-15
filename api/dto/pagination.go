package dto

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/apperror"
)

// PaginationQuery represents pagination parameters for list requests
type PaginationQuery struct {
	Limit      int    `validate:"required,min=1,max=100"`
	Offset     int    `validate:"min=0"`
	Key        string `validate:"omitempty,base64"`
	Reverse    bool   `validate:"omitempty"`
	CountTotal bool   `validate:"omitempty"`
}

func PaginationFromQuery(c *fiber.Ctx) (*PaginationQuery, error) {
	p := &PaginationQuery{
		Limit:      10, // default
		Offset:     0,  // default
		Reverse:    true,
		CountTotal: false,
	}

	// Parse limit
	if limitStr := c.Query("pagination.limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return nil, apperror.NewBadRequest("Limit must be in integer format")
		}

		if limit < 1 || limit > 100 {
			return nil, apperror.NewBadRequest("Limit must be between 1 and 100")
		}

		p.Limit = limit
	}

	// Parse offset
	if offsetStr := c.Query("pagination.offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, apperror.NewBadRequest("Offset must be in integer format")
		}
		p.Offset = offset
	}

	// Parse key
	p.Key = c.Query("pagination.key")

	// Parse reverse
	if reverseStr := c.Query("pagination.reverse"); reverseStr != "" {
		p.Reverse = reverseStr == "true"
	}

	// Parse count_total
	if countTotalStr := c.Query("pagination.count_total"); countTotalStr != "" {
		p.CountTotal = countTotalStr == "true"
	}

	return p, nil
}

// PaginationResponse represents pagination metadata in list responses
type PaginationResponse struct {
	NextKey *string `json:"next_key"`
	Total   int64   `json:"total"`
}
