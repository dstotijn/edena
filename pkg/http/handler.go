package http

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
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
	srv.logger.Info("Captured incoming request.",
		zap.String("host", r.Host),
		zap.String("url", r.URL.String()),
	)

	fmt.Fprintf(w, "host: %v, url: %v", r.Host, r.URL.String())
}
