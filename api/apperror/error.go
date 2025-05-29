package apperror

// Response represents a standardized error response
type Response struct {
	Code    int    `json:"status_code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *Response) Error() string {
	return e.Message
}

// Common error codes
const (
	ErrCodeNotFound     = 404
	ErrCodeBadRequest   = 400
	ErrCodeInternal     = 500
	ErrCodeUnauthorized = 401
)

// Common error messages
const (
	ErrMsgNotFound     = "Resource not found"
	ErrMsgBadRequest   = "Invalid request"
	ErrMsgInternal     = "Internal server error"
	ErrMsgUnauthorized = "Unauthorized"
)

// NewResponse creates a new error response
func NewResponse(code int, message string, details string) *Response {
	if details != "" {
		message = message + ": " + details
	}
	return &Response{
		Code:    code,
		Message: message,
	}
}

// NewNotFound creates a not found error response
func NewNotFound(details string) *Response {
	return NewResponse(ErrCodeNotFound, ErrMsgNotFound, details)
}

// NewBadRequest creates a bad request error response
func NewBadRequest(details string) *Response {
	return NewResponse(ErrCodeBadRequest, ErrMsgBadRequest, details)
}

// NewInternal creates an internal server error response
func NewInternal(details string) *Response {
	return NewResponse(ErrCodeInternal, ErrMsgInternal, details)
}

// NewUnauthorized creates an unauthorized error response
func NewUnauthorized(details string) *Response {
	return NewResponse(ErrCodeUnauthorized, ErrMsgUnauthorized, details)
}

// HandleError transforms any error into a standardized error response
func HandleError(err error) *Response {
	// If it's already our custom error type, return it as is
	if resp, ok := err.(*Response); ok {
		return resp
	}

	// For other errors, wrap them in an internal error
	return NewInternal(err.Error())
}
