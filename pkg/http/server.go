package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"github.com/caddyserver/certmagic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Server represents a server for HTTP and TLS.
type Server struct {
	acmeManager *certmagic.ACMEManager
	httpAddr    string
	tlsAddr     string
	tlsDisabled bool
	tlsConfig   *tls.Config
	logger      *zap.Logger
}

type ServerOption func(*Server)

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		logger: zap.NewNop(),
	}

	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

// WithHTTPAddr overrides the default TCP address for the HTTP server to listen on.
func WithHTTPAddr(addr string) ServerOption {
	return func(srv *Server) {
		srv.httpAddr = addr
	}
}

// WithTLSAddr overrides the default TCP address for the TLS server to listen on.
func WithTLSAddr(addr string) ServerOption {
	return func(srv *Server) {
		srv.tlsAddr = addr
	}
}

// WithACMEManager overrides the ACME manager used. If you call this function
// with `nil`, it will disable ACME support.
func WithACMEManager(am *certmagic.ACMEManager) ServerOption {
	return func(srv *Server) {
		srv.acmeManager = am
	}
}

// WithTLSConfig overrides the TLS config used.
func WithTLSConfig(tlsConfig *tls.Config) ServerOption {
	return func(srv *Server) {
		srv.tlsConfig = tlsConfig
	}
}

// WithoutTLS disables binding on a port for serving TLS. This will implicitly
// disable the TLS-ALPN challenge of the ACME protocol.
func WithoutTLS() ServerOption {
	return func(srv *Server) {
		srv.tlsDisabled = true
	}
}

// WithLogger provides a logger, which is used for HTTP related logs.
func WithLogger(logger *zap.Logger) ServerOption {
	return func(srv *Server) {
		srv.logger = logger
	}
}

// Run starts the HTTP and (if enabled) TLS server.
func (srv *Server) Run(ctx context.Context) error {
	handler := srv.Handler()

	// Start HTTP server.
	go func() {
		err := http.ListenAndServe(srv.httpAddr, handler)
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error: HTTP server failed: %v", err)
		}
	}()

	if !srv.tlsDisabled {
		// Configure TLS server.
		httpServer := &http.Server{
			Addr:      srv.tlsAddr,
			Handler:   handler,
			TLSConfig: srv.tlsConfig,
		}
		if srv.logger != nil {
			logger, err := zap.NewStdLogAt(srv.logger, zapcore.DebugLevel)
			if err != nil {
				return fmt.Errorf("http: failed to create logger: %w", err)
			}
			httpServer.ErrorLog = logger
		}

		// Start TLS server.
		err := httpServer.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http: TLS server failed: %w", err)
		}
	}

	return nil
}
