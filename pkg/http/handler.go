package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/oklog/ulid"
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

	apiRouter := r.MatcherFunc(func(req *http.Request, match *mux.RouteMatch) bool {
		hostname, _ := os.Hostname()
		host, _, _ := net.SplitHostPort(req.Host)
		return strings.EqualFold(host, hostname) || (req.Host == srv.hostname || req.Host == "localhost:8080")
	}).PathPrefix("/api").Subrouter().StrictSlash(true)
	apiRouter.Methods("POST").Path("/hosts/").HandlerFunc(srv.CreateHosts)
	apiRouter.Methods("GET").Path("/http-logs/").HandlerFunc(srv.ListHTTPLogEntries)

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

func parseHostIDs(rawIDs []string) ([]ulid.ULID, *APIError) {
	if len(rawIDs) == 0 {
		return nil, &APIError{
			Message:    "At least one `hostId` query parameter is required.",
			StatusCode: http.StatusBadRequest,
		}
	}
	if len(rawIDs) > 20 {
		return nil, &APIError{
			Message:    "Cannot filter by more than 10 host IDs.",
			StatusCode: http.StatusBadRequest,
		}
	}

	// Use a map for deduplicate behaviour.
	hostIDMap := map[ulid.ULID]interface{}{}
	for _, rawID := range rawIDs {
		hostID, err := ulid.Parse(rawID)
		if err != nil {
			return nil, &APIError{
				Message:    fmt.Sprintf("Failed to parse host ID: %v", err),
				StatusCode: http.StatusBadRequest,
				Err:        err,
			}
		}
		hostIDMap[hostID] = nil
	}

	var hostIDs = make([]ulid.ULID, 0, len(hostIDMap))
	for hostID := range hostIDMap {
		hostIDs = append(hostIDs, hostID)
	}

	return hostIDs, nil
}

type httpLogEntry struct {
	ID        ulid.ULID    `json:"id"`
	HostID    ulid.ULID    `json:"hostId"`
	Request   httpRequest  `json:"request"`
	Response  httpResponse `json:"response"`
	CreatedAt time.Time    `json:"createdAt"`
}

type httpRequest struct {
	Host    string      `json:"host"`
	URL     string      `json:"url"`
	Method  string      `json:"method"`
	Headers http.Header `json:"headers"`
	Body    []byte      `json:"body"`
	Raw     []byte      `json:"raw"`
}

type httpResponse struct {
	StatusCode int         `json:"statusCode"`
	Status     string      `json:"status"`
	Headers    http.Header `json:"headers"`
	Body       []byte      `json:"body"`
	Raw        []byte      `json:"raw"`
}

func (srv *Server) ListHTTPLogEntries(w http.ResponseWriter, r *http.Request) {
	hostIDs, apiErr := parseHostIDs(r.URL.Query()["hostId"])
	if apiErr != nil {
		writeAPIError(w, apiErr)
		return
	}

	params := hosts.ListHTTPLogEntriesParams{
		HostIDs: hostIDs,
	}

	logEntries, err := srv.hostsService.ListHTTPLogEntries(r.Context(), params)
	if err != nil {
		srv.logger.Error("Failed to list HTTP logs.", zap.Error(err))
		srv.handleInternalError(w)
		return
	}

	data := make([]httpLogEntry, len(logEntries))
	for i, logEntry := range logEntries {
		l, err := parseHTTPLogEntry(logEntry)
		if err != nil {
			srv.logger.Error("Failed to parse HTTP log entry.", zap.Error(err))
			srv.handleInternalError(w)
			return
		}
		data[i] = l
	}

	writeAPIResponse(w, APIResponse{
		StatusCode: http.StatusOK,
		Data:       data,
	})
}

func parseHTTPLogEntry(log hosts.HTTPLogEntry) (httpLogEntry, error) {
	reqReader := bufio.NewReader(bytes.NewReader(log.RawRequest))
	req, err := http.ReadRequest(reqReader)
	if err != nil {
		return httpLogEntry{}, fmt.Errorf("failed to read request: %w", err)
	}

	resReader := bufio.NewReader(bytes.NewReader(log.RawResponse))
	res, err := http.ReadResponse(resReader, req)
	if err != nil {
		return httpLogEntry{}, fmt.Errorf("failed to read response: %w", err)
	}

	reqBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return httpLogEntry{}, fmt.Errorf("failed to read request body: %w", err)
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return httpLogEntry{}, fmt.Errorf("failed to read response body: %w", err)
	}

	return httpLogEntry{
		ID:     log.ID,
		HostID: log.HostID,
		Request: httpRequest{
			Host:    req.Host,
			URL:     req.URL.String(),
			Method:  req.Method,
			Headers: req.Header,
			Body:    reqBody,
			Raw:     log.RawRequest,
		},
		Response: httpResponse{
			StatusCode: res.StatusCode,
			Status:     res.Status,
			Headers:    res.Header,
			Body:       resBody,
			Raw:        log.RawResponse,
		},
		CreatedAt: ulid.Time(log.ID.Time()).UTC(),
	}, nil
}
