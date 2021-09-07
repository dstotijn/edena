package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/caddyserver/certmagic"
	badgerdb "github.com/dgraph-io/badger/v3"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/dstotijn/edena/pkg/database/badger"
	"github.com/dstotijn/edena/pkg/dns"
	"github.com/dstotijn/edena/pkg/hosts"
	"github.com/dstotijn/edena/pkg/http"
)

var (
	hostname    string
	httpAddr    string
	tlsAddr     string
	dnsAddr     string
	prettyPrint bool
)

func init() {
	rootCmd.AddCommand(serverCmd)
	osHostname, _ := os.Hostname()
	serverCmd.Flags().StringVarP(&hostname, "hostname", "H", osHostname, "hostname used for wildcard certificate and base for subdomains")
	serverCmd.Flags().StringVar(&httpAddr, "http", ":80",
		`the TCP address for the HTTP server to listen on, in the form "host:port"`)
	serverCmd.Flags().StringVar(&tlsAddr, "tls", ":443",
		`the TCP address for the HTTPS server to listen on, in the form "host:port"`)
	serverCmd.Flags().StringVar(&dnsAddr, "dns", ":53",
		`the address for the DNS server to listen on, in the form "host:port"`)
	serverCmd.Flags().BoolVar(&prettyPrint, "pretty-print", false, "use pretty log formatting")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs a server for collecting and managing HTTP, SMTP and DNS traffic.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()

		logger, err := newLogger(debug, prettyPrint)
		if err != nil {
			return err
		}
		defer logger.Sync()
		serverLogger := logger.Named("server")

		dataDir, err := dataDirectory()
		if err != nil {
			return fmt.Errorf("failed to configure data directory: %w", err)
		}

		// Storage is used for certificates and ACME DNS-01 challenge records.
		// For now, it's hardcoded to file storage, but eventually we'll offer
		// other types of database/repositories as well.
		storage := &certmagic.FileStorage{Path: dataDir}

		// Configre a dns.Server, which is used for capturing DNS requests,
		// and solving ACME DNS-01 challenges.
		dnsServer := dns.NewServer(
			dns.WithStorage(storage),
			dns.WithAddress(dnsAddr),
			dns.WithSOAHostname(hostname),
			dns.WithLogger(logger.Named("dns")),
		)

		// Configure default ACME manager for certificates.
		certmagicLogger := logger.Named("certmagic")
		certmagicConfig := certmagic.NewDefault()
		certmagicConfig.Storage = storage
		certmagicConfig.Logger = certmagicLogger

		acmeManager := certmagic.NewACMEManager(certmagicConfig, certmagic.ACMEManager{
			Logger: certmagicLogger,
			DNS01Solver: &certmagic.DNS01Solver{
				DNSProvider: dnsServer,
			},
		})

		certmagicConfig.Issuers = []certmagic.Issuer{acmeManager}

		tlsConfig := certmagicConfig.TLSConfig()

		dbPath := path.Join(dataDir, "db")
		dbLogger := logger.WithOptions(zap.IncreaseLevel(zapcore.WarnLevel)).
			Named("database").
			Sugar()

		db, err := badger.OpenDatabase(
			badgerdb.DefaultOptions(dbPath).WithLogger(badger.NewLogger(dbLogger)),
		)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				logger.Error("Failed to close database.", zap.Error(err))
			}
		}()

		// Configure hosts.Service, which is used to maintain hosts and store
		// network interactions.
		hostsService := hosts.NewService(
			hosts.WithBaseHostname(hostname),
			hosts.WithDatabase(db),
			hosts.WithLogger(logger.Named("hosts")),
		)

		// Configure an http.Server, which orchestrates running HTTP and HTTPS servers.
		// We're use HTTP and TLS for:
		// - Capturing requests
		// - API and Web UI
		// - Solving ACME challenges (HTTP-01 and TLS-ALPN)
		httpServer := http.NewServer(
			http.WithHostname(hostname),
			http.WithACMEManager(acmeManager),
			http.WithTLSConfig(tlsConfig),
			http.WithHTTPAddr(httpAddr),
			http.WithTLSAddr(tlsAddr),
			http.WithHostsService(hostsService),
			http.WithLogger(logger.Named("http")),
		)

		serverLogger.Info("Running Edena ...",
			zap.String("hostname", hostname),
			zap.Bool("debug", debug),
		)

		go func() {
			if err := dnsServer.Run(ctx); err != nil {
				log.Fatalf("Failed to run DNS server: %v", err)
			}
		}()

		go func() {
			err := certmagicConfig.ManageAsync(ctx, []string{hostname, "*." + hostname})
			if err != nil {
				log.Printf("Failed to obtain wildcard certificate: %v", err)
			}
		}()

		go func() {
			if err := httpServer.Run(ctx); err != nil {
				log.Fatalf("Failed to run HTTP server(s): %v", err)
			}
		}()

		// Wait for interrupt signal.
		<-ctx.Done()
		// Restore signal, allowing "force quit".
		stop()

		serverLogger.Info("Shutting down server. Press Ctrl+C to force quit.")

		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			if err := httpServer.Shutdown(timeoutCtx); err != nil {
				serverLogger.Error("Failed to shutdown HTTP server(s).", zap.Error(err))
			}
			wg.Done()
		}()
		go func() {
			if err := dnsServer.Shutdown(timeoutCtx); err != nil {
				serverLogger.Error("Failed to shutdown DNS server.", zap.Error(err))
			}
			wg.Done()
		}()

		wg.Wait()

		return nil
	},
}

func dataDirectory() (baseDir string, err error) {
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		baseDir, err = homedir.Expand(xdgData)
	} else {
		baseDir, err = homedir.Dir()
	}
	if err != nil {
		return
	}

	baseDir = filepath.Join(baseDir, ".local", "share", "edena")

	return
}
