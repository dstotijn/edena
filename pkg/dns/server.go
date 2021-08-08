package dns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/libdns"
	"github.com/miekg/dns"
)

// Interface guards.
var (
	_ certmagic.ACMEDNSProvider = (*Server)(nil)
	_ libdns.RecordGetter       = (*Server)(nil)
)

var (
	ErrRecordAlreadyExists = errors.New("dns record already exists")
	ErrRecordNotFound      = errors.New("dns record not found")
)

// Server is used for capturing DNS requests, and storing/serving TXT records
// for the ACME DNS-01 challenge. It implements certmagic.ACMEDNSProvider.
type Server struct {
	storage     certmagic.Storage
	addr        string
	soaHostname string
}

type ServerOption func(*Server)

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		addr: ":53",
	}

	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

func WithStorage(s certmagic.Storage) ServerOption {
	return func(srv *Server) {
		srv.storage = s
	}
}

func WithAddress(addr string) ServerOption {
	return func(srv *Server) {
		srv.addr = addr
	}
}

func WithSOAHostname(soaHostname string) ServerOption {
	return func(srv *Server) {
		srv.soaHostname = dns.Fqdn(soaHostname)
	}
}

// Run starts the DNS server.
func (srv *Server) Run(ctx context.Context) error {
	go func() {
		dnsServer := &dns.Server{
			Addr:      srv.addr,
			Net:       "udp",
			Handler:   srv,
			ReusePort: true,
		}
		if err := dnsServer.ListenAndServe(); err != nil {
			log.Fatalf("dns: server failed: %v", err)
		}
	}()

	dnsServer := &dns.Server{
		Addr:      srv.addr,
		Net:       "tcp",
		Handler:   srv,
		ReusePort: true,
	}

	if err := dnsServer.ListenAndServe(); err != nil {
		return fmt.Errorf("dns: server failed: %w", err)
	}

	return nil
}

func lockKey(zone string) string {
	return "dns:" + strings.TrimSuffix(zone, ".")
}

func storageKey(zone string) string {
	zoneKey := strings.TrimSuffix(dns.CanonicalName(zone), ".")
	return path.Join("dns", zoneKey)
}

func (srv *Server) AppendRecords(ctx context.Context, zone string, newRecs []libdns.Record) ([]libdns.Record, error) {
	var recs []libdns.Record
	var createdRecords []libdns.Record

	lockKey := lockKey(zone)
	if err := srv.storage.Lock(ctx, lockKey); err != nil {
		return nil, fmt.Errorf("dns: failed to obtain lock: %w", err)
	}
	defer func() {
		if err := srv.storage.Unlock(lockKey); err != nil {
			log.Printf("Error: Failed to unlock key %q: %v", lockKey, err)
		}
	}()

	storageKey := storageKey(zone)

	zonefile, err := srv.storage.Load(storageKey)
	var errNotExist certmagic.ErrNotExist
	// Absorb `certmagic.ErrNotExist`, but return all other errors.
	if err != nil && !errors.As(err, &errNotExist) {
		return nil, fmt.Errorf("dns: failed to load zonefile from storage: %w", err)
	}

	if zonefile != nil {
		err = json.Unmarshal(zonefile, &recs)
		if err != nil {
			return nil, fmt.Errorf("dns: failed to decode zonefile JSON: %w", err)
		}
	}

Loop:
	for _, newRec := range newRecs {
		for _, rec := range recs {
			if (newRec.ID != "" && newRec.ID == rec.ID) || (newRec.Name == rec.Name && newRec.Type == rec.Type) {
				continue Loop
			}
		}
		recs = append(recs, newRec)
		createdRecords = append(createdRecords, newRec)
	}

	newZonefile, err := json.Marshal(recs)
	if err != nil {
		return nil, fmt.Errorf("dns: failed to encode zonefile JSON: %w", err)
	}

	err = srv.storage.Store(storageKey, newZonefile)
	if err != nil {
		return nil, fmt.Errorf("dns: failed to store zonefile (key: %q): %w", storageKey, err)
	}

	return createdRecords, nil
}

func (srv *Server) DeleteRecords(ctx context.Context, zone string, deleteRecs []libdns.Record) ([]libdns.Record, error) {
	var recs []libdns.Record
	var deletedRecs []libdns.Record

	lockKey := lockKey(zone)
	if err := srv.storage.Lock(ctx, lockKey); err != nil {
		return nil, fmt.Errorf("dns: failed to obtain lock: %w", err)
	}
	defer func() {
		if err := srv.storage.Unlock(lockKey); err != nil {
			log.Printf("Error: Failed to unlock key %q: %v", lockKey, err)
		}
	}()

	storageKey := storageKey(zone)

	zonefile, err := srv.storage.Load(storageKey)
	var errNotExist *certmagic.ErrNotExist
	// Absorb `certmagic.ErrNotExist`, but return all other errors.
	if err != nil && !errors.As(err, errNotExist) {
		return nil, fmt.Errorf("dns: failed to load zonefile from storage: %w", err)
	}

	if zonefile != nil {
		err = json.Unmarshal(zonefile, &recs)
		if err != nil {
			return nil, fmt.Errorf("dns: failed to decode zonefile JSON: %w", err)
		}
	}

	// Filter out existing records that need to be deleted.
	filteredRecs := recs[:0]
	for _, deleteRec := range deleteRecs {
		for _, rec := range recs {
			if (deleteRec.ID != "" && deleteRec.ID == rec.ID) || (deleteRec.Name == rec.Name && deleteRec.Type == rec.Type) {
				deletedRecs = append(deletedRecs, deleteRec)
			} else {
				filteredRecs = append(filteredRecs, rec)
			}
		}
	}

	newZonefile, err := json.Marshal(filteredRecs)
	if err != nil {
		return nil, fmt.Errorf("dns: failed to encode zonefile JSON: %w", err)
	}

	err = srv.storage.Store(storageKey, newZonefile)
	if err != nil {
		return nil, fmt.Errorf("dns: failed to store zonefile (storage key: %q): %w", storageKey, err)
	}

	return deletedRecs, nil
}

func (srv *Server) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	var recs []libdns.Record

	lockKey := lockKey(zone)
	if err := srv.storage.Lock(ctx, lockKey); err != nil {
		return nil, fmt.Errorf("dns: failed to obtain lock: %w", err)
	}
	defer func() {
		if err := srv.storage.Unlock(lockKey); err != nil {
			log.Printf("Error: Failed to unlock key %q: %v", lockKey, err)
		}
	}()

	storageKey := storageKey(zone)
	zonefile, err := srv.storage.Load(storageKey)
	var errNotExist certmagic.ErrNotExist
	if errors.As(err, &errNotExist) {
		return recs, nil
	}
	if err != nil {
		return nil, fmt.Errorf("dns: failed to get zonefile (storage key: %q): %w", storageKey, err)
	}

	err = json.Unmarshal(zonefile, &recs)
	if err != nil {
		return nil, fmt.Errorf("dns: failed to decode zonefile JSON: %w", err)
	}

	return recs, nil
}

// MessageFromRecord parses a libdns.Record and returns a dns.Msg value, using
// the `zone` argument.
func MessageFromRecord(zone string, rec libdns.Record) (dns.RR, error) {
	var rr dns.RR

	rrType, ok := dns.StringToType[rec.Type]
	if !ok {
		return nil, fmt.Errorf("dns: unknown record type %q", rec.Type)
	}

	switch rrType {
	case dns.TypeTXT:
		rr = &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   libdns.AbsoluteName(rec.Name, zone),
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    3600,
			},
			Txt: []string{rec.Value},
		}
	default:
		return nil, fmt.Errorf("dns: unsupported record type %q", dns.TypeToString[rrType])
	}

	return rr, nil
}
