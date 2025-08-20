package apperror

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Response represents a standardized error response
type Response struct {
	Code    ErrorCode `json:"status_code"`
	Message string    `json:"message"`
}

// Error implements the error interface
func (e *Response) Error() string {
	return e.Message
}

type ErrorCode int

// Common error codes
const (
	ErrCodeBadRequest   ErrorCode = 400
	ErrCodeUnauthorized ErrorCode = 401
	ErrCodeNotFound     ErrorCode = 404
	ErrCodeInternal     ErrorCode = 500
)

// NewResponse creates a new error response
func NewResponse(code ErrorCode, message string) *Response {
	return &Response{
		Code:    code,
		Message: message,
	}
}

// NewNotFound creates a not found error response
func NewNotFound() *Response {
	return NewResponse(ErrCodeNotFound, ErrMsgNotFound)
}

// NewBadRequest creates a bad request error response
func NewBadRequest() *Response {
	return NewResponse(ErrCodeBadRequest, ErrMsgBadRequest)
}

// NewValidationError creates a validation error response
func NewValidationError(message string) *Response {
	return NewResponse(ErrCodeBadRequest, message)
}

// NewInternal creates an internal server error response
func NewInternal() *Response {
	return NewResponse(ErrCodeInternal, ErrMsgInternal)
}

// NewUnauthorized creates an unauthorized error response
func NewUnauthorized() *Response {
	return NewResponse(ErrCodeUnauthorized, ErrMsgUnauthorized)
}

// NewDuplicateStatus creates a duplicate status error response
func NewDuplicateStatus() *Response {
	return NewResponse(ErrCodeBadRequest, ErrMsgDuplicateStatus)
}

// NewTxNotFound creates a transaction not found error response
func NewTxNotFound(hash string) *Response {
	return NewResponse(ErrCodeNotFound, fmt.Sprintf(ErrMsgTxNotFound, hash))
}

// NewNoValidTxFiles creates a no valid transaction files error response
func NewNoValidTxFiles(hash string) *Response {
	return NewResponse(ErrCodeNotFound, fmt.Sprintf(ErrMsgNoValidTxFiles, hash))
}

// NewInvalidLimit creates an invalid limit error response
func NewInvalidLimit() *Response {
	return NewResponse(ErrCodeBadRequest, ErrMsgInvalidLimit)
}

// NewLimitInteger creates a limit integer error response
func NewLimitInteger() *Response {
	return NewResponse(ErrCodeBadRequest, ErrMsgLimitInteger)
}

// NewOffsetInteger creates an offset integer error response
func NewOffsetInteger() *Response {
	return NewResponse(ErrCodeBadRequest, ErrMsgOffsetInteger)
}

// NewReverse creates a reverse boolean error response
func NewReverse() *Response {
	return NewResponse(ErrCodeBadRequest, ErrMsgReverse)
}

// NewCountTotal creates a count total boolean error response
func NewCountTotal() *Response {
	return NewResponse(ErrCodeBadRequest, ErrMsgCountTotal)
}

// handleError transforms any error into a standardized error response
func handleError(err error) *Response {
	// If it's already our custom error type, return it as is
	if resp, ok := err.(*Response); ok {
		return resp
	}

	// Handle GORM not found
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return NewNotFound()
	}

	// For other errors, wrap them in an internal error
	return NewInternal()
}

// HandleErrorResponse is a helper function that handles error responses in a standardized way
// It takes an error, converts it to a Response, and returns the appropriate HTTP response
func HandleErrorResponse(c *fiber.Ctx, err error) error {
	errResp := handleError(err)
	return c.Status(errResp.Code.Int()).JSON(errResp)
}

// Int returns the error code as an int
func (e ErrorCode) Int() int {
	return int(e)
}
