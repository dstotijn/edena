package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/oklog/ulid"
	"go.uber.org/zap"
)

func (srv *Server) Handler() http.Handler {
	r := mux.NewRouter()

	if srv.acmeManager != nil {
		// Register ACME HTTP-01 challenge middleware.
		r.Use(srv.acmeManager.HTTPChallengeHandler)
	}

	r.PathPrefix("").HandlerFunc(srv.CaptureRequest)

	return r
}

func (srv Server) CaptureRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now().UTC()
	id, close := srv.mustNewULID(now)
	defer close()

	entry := LogEntry{
		ID:     id,
		HostID: ulid.MustParse("0000XSNJG0MQJHBF4QX1EFD6Y3"), // TODO: Replace dummy placeholder
	}

	err := srv.database.StoreHTTPLogEntry(ctx, entry)
	if err != nil {
		srv.logger.Error("Failed to store HTTP log entry.", zap.Error(err))
		code := http.StatusInternalServerError
		http.Error(w, http.StatusText(code), code)
		return
	}

	srv.logger.Info("Captured incoming request.",
		zap.String("host", r.Host),
		zap.String("url", r.URL.String()),
	)

	fmt.Fprintf(w, "host: %v, url: %v", r.Host, r.URL.String())
}
