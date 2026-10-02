package httpx

import "net/http"

// APIError is an error that is safe to return to the client.
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (e *APIError) Error() string { return e.Message }

func NewError(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

func BadRequest(message string) *APIError {
	return NewError(http.StatusBadRequest, "bad_request", message)
}

func NotFound(message string) *APIError {
	return NewError(http.StatusNotFound, "not_found", message)
}

func Conflict(code, message string, details any) *APIError {
	return &APIError{Status: http.StatusConflict, Code: code, Message: message, Details: details}
}
