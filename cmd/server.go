package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/caddyserver/certmagic"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/dstotijn/edena/pkg/dns"
	"github.com/dstotijn/edena/pkg/http"
)

var (
	hostname string
	httpAddr string
	tlsAddr  string
	dnsAddr  string
)

func init() {
	rootCmd.AddCommand(serverCmd)
	osHostname, _ := os.Hostname()
	serverCmd.Flags().StringVarP(&hostname, "hostname", "H", osHostname, "hostname used for wildcard certificate and base for subdomains")
	serverCmd.Flags().StringVar(&httpAddr, "http", ":80",
		`the TCP address for the HTTP server to listen on, in the form "host:port"`)
	serverCmd.Flags().StringVar(&tlsAddr, "tls", ":443",
		`the TCP address for the TLS server to listen on, in the form "host:port"`)
	serverCmd.Flags().StringVar(&dnsAddr, "dns", ":53",
		`the address for the DNS server to listen on, in the form "host:port"`)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs a server for collecting and managing HTTP, SMTP and DNS traffic.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		config := zap.NewDevelopmentConfig()
		config.Level.SetLevel(zap.InfoLevel)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logger, _ := config.Build()
		defer logger.Sync()

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
		)

		// Configure default ACME manager for certificates.
		certmagicConfig := certmagic.NewDefault()
		certmagicConfig.Storage = storage
		certmagicConfig.Logger = logger

		acmeManager := certmagic.NewACMEManager(certmagicConfig, certmagic.ACMEManager{
			Logger: logger,
			DNS01Solver: &certmagic.DNS01Solver{
				DNSProvider: dnsServer,
			},
		})

		certmagicConfig.Issuers = append(certmagicConfig.Issuers, acmeManager)

		tlsConfig := certmagicConfig.TLSConfig()

		// Configure an http.Server, which orchestrates running HTTP and TLS servers.
		// We're use HTTP and TLS for:
		// - Capturing requests
		// - API and Web UI
		// - Solving ACME challenges (HTTP-01 and TLS-ALPN)
		httpServer := http.NewServer(
			http.WithACMEManager(acmeManager),
			http.WithTLSConfig(tlsConfig),
			http.WithHTTPAddr(httpAddr),
			http.WithTLSAddr(tlsAddr),
			http.WithLogger(logger.Named("http")),
		)

		go func() {
			if err := dnsServer.Run(ctx); err != nil {
				log.Fatal(err)
			}
		}()

		go func() {
			err := certmagicConfig.ManageAsync(ctx, []string{hostname, "*." + hostname})
			if err != nil {
				log.Printf("Failed to obtain wildcard certificate: %v", err)
			}
		}()

		if err := httpServer.Run(ctx); err != nil {
			return err
		}

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
