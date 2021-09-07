package http

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Data       interface{} `json:"data,omitempty"`
	Error      *APIError   `json:"error,omitempty"`
	StatusCode int         `json:"-"`
}

type APIError struct {
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	Err        error  `json:"-"`
}

func (e *APIError) Error() string {
	return e.Message
}

func (e *APIError) Unwrap() error {
	return e.Err
}

func writeAPIError(w http.ResponseWriter, err *APIError) {
	writeAPIResponse(w, APIResponse{
		Error:      err,
		StatusCode: err.StatusCode,
	})
}

func writeAPIResponse(w http.ResponseWriter, res APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.StatusCode)
	_ = json.NewEncoder(w).Encode(res)
}

func (srv *Server) handleInternalError(w http.ResponseWriter) {
	writeAPIError(w, &APIError{
		Message:    "Internal server error. Please try again.",
		StatusCode: http.StatusInternalServerError,
	})
}
