package dto

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/core-indexer/api/apperror"
)

// PaginationQuery represents pagination parameters for list requests
type PaginationQuery struct {
	Limit      int    `validate:"required,min=1,max=1000"`
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
		CountTotal: true,
	}

	// Parse limit
	if limitStr := c.Query("pagination.limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return nil, apperror.NewLimitInteger()
		}

		if limit < 1 || limit > 1000 {
			return nil, apperror.NewInvalidLimit()
		}

		p.Limit = limit
	}

	// Parse offset
	if offsetStr := c.Query("pagination.offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, apperror.NewOffsetInteger()
		}

		if offset < 0 {
			return nil, apperror.NewOffsetInteger()
		}

		p.Offset = offset
	}

	// Parse key
	key := c.Query("pagination.key")
	if key != "" {
		if err := parsePaginationKey(key, p); err != nil {
			return nil, err
		}
	}

	// Parse reverse
	if reverseStr := c.Query("pagination.reverse"); reverseStr != "" {
		val, err := strconv.ParseBool(reverseStr)
		if err != nil {
			return nil, apperror.NewReverse()
		}
		p.Reverse = val

	}

	// Parse count_total
	if countTotalStr := c.Query("pagination.count_total"); countTotalStr != "" {
		val, err := strconv.ParseBool(countTotalStr)
		if err != nil {
			return nil, apperror.NewCountTotal()
		}
		p.CountTotal = val
	}

	return p, nil
}

// PaginationResponse represents pagination metadata in list responses
type PaginationResponse struct {
	PreviousKey *string `json:"previous_key" `
	NextKey     *string `json:"next_key"`
	Total       string  `json:"total"`
}

func NewPaginationResponse(offset int, limit int, total int64) (res PaginationResponse) {
	if total == -1 || total > int64(offset+limit) {
		nextKey := base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset + limit)))
		res.NextKey = &nextKey
	}
	// if offset is greater than or equal to limit, previousKey can be set
	if offset > 0 && offset >= limit {
		previousKey := base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset - limit)))
		res.PreviousKey = &previousKey
	}
	res.Total = fmt.Sprintf("%d", total)
	return
}

// parsePaginationKey parses the pagination key and updates pagination accordingly
func parsePaginationKey(key string, pagination *PaginationQuery) error {
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return errors.New("pagination.key must be a valid base64 encoded string")
	}

	// Try JSON cursor format first
	if strings.HasPrefix(string(bytes.TrimSpace(decoded)), "{") {
		return parseJSONCursor(decoded, pagination)
	}

	// Fallback to traditional integer approach
	return parseIntegerCursor(string(decoded), pagination)
}

// parseJSONCursor attempts to parse JSON cursor format
func parseJSONCursor(decoded []byte, pagination *PaginationQuery) error {
	// JSON parsing failed, try fallback to integer with appropriate error message
	parsedOffset, err := strconv.Atoi(string(decoded))
	if err != nil || parsedOffset < 0 {
		return errors.New("invalid pagination.key format")
	}

	pagination.Offset = parsedOffset
	return nil
}

// parseIntegerCursor parses traditional integer cursor format
func parseIntegerCursor(decodedStr string, pagination *PaginationQuery) error {
	parsedOffset, err := strconv.Atoi(decodedStr)
	if err != nil || parsedOffset < 0 {
		return errors.New("pagination.key must decode to a nonnegative integer")
	}

	pagination.Offset = parsedOffset
	return nil
}
