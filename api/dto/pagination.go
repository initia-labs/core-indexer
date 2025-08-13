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
		parsedLimit, err := strconv.ParseInt(limitStr, 10, 64)

		if err != nil {
			return nil, err
		}

		if parsedLimit > 100 {
			return nil, apperror.NewBadRequest("pagination.limit cannot exceed 100")
		}
	}

	// Parse offset
	if offsetStr := c.Query("pagination.offset"); offsetStr != "" {
		if parsedOffset, err := strconv.ParseInt(offsetStr, 10, 64); err == nil {
			p.Offset = int(parsedOffset)
		}
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
