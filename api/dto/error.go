package dto

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}
