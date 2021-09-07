package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/dstotijn/edena/pkg/hosts"
)

func (srv *Server) Handler() http.Handler {
	r := mux.NewRouter()
	r.Use(srv.RecoveryMiddleware)

	if srv.acmeManager != nil {
		// Register ACME HTTP-01 challenge middleware.
		r.Use(srv.acmeManager.HTTPChallengeHandler)
	}

	apiRouter := r.Host(srv.hostname).PathPrefix("/api").Subrouter().StrictSlash(true)
	apiRouter.Methods("POST").Path("/hosts").HandlerFunc(srv.CreateHosts)

	r.PathPrefix("").HandlerFunc(srv.CaptureRequest)

	return r
}

func (srv *Server) RecoveryMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				srv.logger.Sugar().Errorf("Recovered from panic: %v", err)
				srv.handleInternalError(w)
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func (srv *Server) CaptureRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := srv.hostsService.StoreHTTPLogEntry(ctx, hosts.StoreHTTPLogEntryParams{
		Request:  r,
		Response: &http.Response{},
	})
	if errors.Is(err, hosts.ErrHostNotFound) {
		srv.logger.Info("Host not found, ignorning incoming request.", zap.Error(err))
		code := http.StatusNotFound
		http.Error(w, http.StatusText(code), code)
		return
	}
	if err != nil {
		srv.logger.Error("Failed to store HTTP log entry.", zap.Error(err))
		srv.handleInternalError(w)
		return
	}

	fmt.Fprint(w, "OK")
}

type createHostRequestBody struct {
	Amount int `json:"amount"`
}

func (body *createHostRequestBody) validate() *APIError {
	if body.Amount < 1 || body.Amount > 50 {
		return &APIError{
			Message:    `Property "amount" must be min 1, max 50.`,
			StatusCode: http.StatusBadRequest,
		}
	}
	return nil
}

func (srv *Server) CreateHosts(w http.ResponseWriter, r *http.Request) {
	var body createHostRequestBody

	err := json.NewDecoder(r.Body).Decode(&body)
	if err == io.EOF {
		apiErr := &APIError{
			Message:    "Request body cannot be empty.",
			StatusCode: http.StatusBadRequest,
			Err:        err,
		}
		writeAPIError(w, apiErr)
		return
	}
	if err != nil {
		apiErr := &APIError{
			Message:    fmt.Sprintf("Failed to parse request body: %v", err),
			StatusCode: http.StatusBadRequest,
			Err:        err,
		}
		writeAPIError(w, apiErr)
		return
	}

	if err := body.validate(); err != nil {
		writeAPIError(w, err)
		return
	}

	hosts, err := srv.hostsService.CreateHosts(r.Context(), body.Amount)
	if err != nil {
		srv.logger.Error("Failed to create hosts.", zap.Error(err))
		srv.handleInternalError(w)
		return
	}

	writeAPIResponse(w, APIResponse{
		StatusCode: http.StatusCreated,
		Data:       hosts,
	})
}
