package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"

	"github.com/caddyserver/certmagic"
	"github.com/dstotijn/edena/pkg/hosts"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Server represents a server for HTTP and TLS.
type Server struct {
	hostsService hosts.Service
	hostname     string
	acmeManager  *certmagic.ACMEManager
	httpAddr     string
	tlsAddr      string
	tlsDisabled  bool
	tlsConfig    *tls.Config
	httpServer   *http.Server
	tlsServer    *http.Server
	logger       *zap.Logger
}

type ServerOption func(*Server)

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		httpAddr: ":80",
		tlsAddr:  ":443",
		logger:   zap.NewNop(),
	}

	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

// WithHostsService sets the hosts.Service used for managing hosts, e.g.
// creating new hosts and capturing network traffic received on them.
func WithHostsService(svc hosts.Service) ServerOption {
	return func(srv *Server) {
		srv.hostsService = svc
	}
}

// WithHostname sets the hostname used to serve the API.
func WithHostname(hostname string) ServerOption {
	return func(srv *Server) {
		srv.hostname = hostname
	}
}

// WithHTTPAddr overrides the default TCP address for the HTTP server to listen on.
func WithHTTPAddr(addr string) ServerOption {
	return func(srv *Server) {
		srv.httpAddr = addr
	}
}

// WithTLSAddr overrides the default TCP address for the HTTPS server to listen on.
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

// Run starts the HTTP and (if enabled) HTTPS server.
func (srv *Server) Run(ctx context.Context) error {
	var result *multierror.Error
	var wg sync.WaitGroup
	handler := srv.Handler()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Configure HTTPS server.
		httpServer := &http.Server{
			Addr:    srv.httpAddr,
			Handler: handler,
		}
		if srv.logger != nil {
			logger, err := zap.NewStdLogAt(srv.logger, zapcore.DebugLevel)
			if err != nil {
				srv.logger.Error("Failed to create HTTP logger.", zap.Error(err))
			} else {
				httpServer.ErrorLog = logger
			}
		}
		srv.httpServer = httpServer

		// Start HTTP server.
		srv.logger.Info(fmt.Sprintf("HTTP server listening on %v ...", srv.httpAddr))
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			srv.logger.Error("HTTP server failed.", zap.Error(err))
			result = multierror.Append(result, err)
		}
	}()

	if !srv.tlsDisabled {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Configure HTTPS server.
			tlsServer := &http.Server{
				Addr:      srv.tlsAddr,
				Handler:   handler,
				TLSConfig: srv.tlsConfig,
			}
			if srv.logger != nil {
				logger, err := zap.NewStdLogAt(srv.logger, zapcore.DebugLevel)
				if err != nil {
					srv.logger.Error("Failed to create TLS logger.", zap.Error(err))
				} else {
					tlsServer.ErrorLog = logger
				}
			}
			srv.tlsServer = tlsServer

			// Start HTTPS server.
			srv.logger.Info(fmt.Sprintf("HTTPS server listening on %v ...", srv.tlsAddr))
			err := srv.tlsServer.ListenAndServeTLS("", "")
			if err != nil && err != http.ErrServerClosed {
				srv.logger.Error("HTTPS server failed.", zap.Error(err))
				result = multierror.Append(result, err)
			}
		}()
	}

	wg.Wait()

	if result != nil && len(result.Errors) > 0 {
		return fmt.Errorf("http: failed to run servers: %w", result)
	}

	return nil
}

func (srv *Server) Shutdown(ctx context.Context) error {
	// We don't use the `errgroup` package, because we want to await *all*
	// errors before returning.
	var result *multierror.Error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if srv.httpServer == nil {
			return
		}
		srv.httpServer.SetKeepAlivesEnabled(false)
		if err := srv.httpServer.Shutdown(ctx); err != nil && err != context.DeadlineExceeded {
			srv.logger.Error("Failed to shutdown HTTP server.", zap.Error(err))
			result = multierror.Append(result, err)
		}
	}()
	go func() {
		defer wg.Done()
		if srv.tlsServer == nil {
			return
		}
		srv.tlsServer.SetKeepAlivesEnabled(false)
		if err := srv.tlsServer.Shutdown(ctx); err != nil && err != context.DeadlineExceeded {
			srv.logger.Error("Failed to shutdown HTTPS server.", zap.Error(err))
			result = multierror.Append(result, err)
		}
	}()

	wg.Wait()

	if result != nil && len(result.Errors) > 0 {
		return fmt.Errorf("http: failed to shutdown servers: %w", result)
	}

	return nil
}
