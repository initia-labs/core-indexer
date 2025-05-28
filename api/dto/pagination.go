package dto

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// PaginationQuery represents pagination parameters for list requests
type PaginationQuery struct {
	Limit      int64  `validate:"required,min=1,max=100"`
	Offset     int64  `validate:"min=0"`
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
		if limit, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
			p.Limit = limit
		}
	}

	// Parse offset
	if offsetStr := c.Query("pagination.offset"); offsetStr != "" {
		if offset, err := strconv.ParseInt(offsetStr, 10, 64); err == nil {
			p.Offset = offset
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
