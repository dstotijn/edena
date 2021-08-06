package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/caddyserver/certmagic"
)

// Server represents a server for HTTP and TLS.
type Server struct {
	acmeManager *certmagic.ACMEManager
	httpAddr    string
	tlsAddr     string
	tlsDisabled bool
	tlsConfig   *tls.Config
}

type ServerOption func(*Server)

func NewServer(opts ...ServerOption) *Server {
	// Configure default ACME manager for certificates.
	certmagicConfig := certmagic.NewDefault()
	certmagicConfig.OnDemand = &certmagic.OnDemandConfig{}
	acmeManager := certmagic.NewACMEManager(certmagicConfig, certmagic.DefaultACME)

	tlsConfig := certmagicConfig.TLSConfig()

	srv := &Server{
		acmeManager: acmeManager,
		tlsConfig:   tlsConfig,
		httpAddr:    ":80",
		tlsAddr:     ":443",
	}

	for _, opt := range opts {
		opt(srv)
	}

	return srv
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

	ln, err := net.Listen("tcp", srv.tlsAddr)
	if err != nil {
		log.Fatalf("Error: Failed to create TLS listener: %v", err)
	}
	defer ln.Close()

	if !srv.tlsDisabled {
		// Configure TLS server.
		httpServer := &http.Server{
			Handler:   handler,
			TLSConfig: srv.tlsConfig,
		}
		// Start TLS server.
		err = httpServer.ServeTLS(ln, "", "")
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http: HTTPS server failed: %w", err)
		}
	}

	return nil
}
