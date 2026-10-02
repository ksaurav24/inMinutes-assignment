// Package httpx defines the API's response envelope and helpers to write it.
//
// Success: {"data": ...}
// Error:   {"error": {"code": "...", "message": "..."}}
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type successBody struct {
	Data any `json:"data"`
}

type errorBody struct {
	Error *APIError `json:"error"`
}

// JSON writes data wrapped in the success envelope.
func JSON(w http.ResponseWriter, status int, data any) {
	write(w, status, successBody{Data: data})
}

// Error writes err in the error envelope. An *APIError is sent as is; any
// other error is logged and hidden behind a generic 500 so internals don't leak.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	apiErr, ok := err.(*APIError)
	if !ok {
		slog.ErrorContext(r.Context(), "internal error",
			"method", r.Method,
			"path", r.URL.Path,
			"err", err,
		)
		apiErr = NewError(http.StatusInternalServerError, "internal_error", "internal server error")
	}
	write(w, apiErr.Status, errorBody{Error: apiErr})
}

func write(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode response", "err", err)
	}
}
