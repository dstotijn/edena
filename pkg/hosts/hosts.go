package hosts

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/oklog/ulid"
	"go.uber.org/zap"
)

const hostHashLength = 4

var ErrHostNotFound = errors.New("host not found")

type Host struct {
	ID       ulid.ULID `json:"id"`
	Hostname string    `json:"hostname"`
}

type HTTPLogEntry struct {
	ID          ulid.ULID
	HostID      ulid.ULID
	Request     *http.Request
	Response    *http.Response
	RawRequest  []byte
	RawResponse []byte
}

func (srv *service) CreateHosts(ctx context.Context, amount int) ([]Host, error) {
	hosts := make([]Host, amount)

	for i := 0; i < amount; i++ {
		randBytes := make([]byte, hostHashLength)
		_, err := rand.Read(randBytes)
		if err != nil {
			return nil, fmt.Errorf("hosts: failed to generate random bytes: %w", err)
		}

		hosts[i] = Host{
			ID:       ulid.MustNew(ulid.Timestamp(time.Now()), ulidEntropy),
			Hostname: fmt.Sprintf("%v-%v.%v", petname.Generate(2, "-"), hex.EncodeToString(randBytes), srv.baseHostname),
		}

		err = srv.database.StoreHosts(ctx, hosts...)
		if err != nil {
			return nil, fmt.Errorf("hosts: failed to store hosts: %w", err)
		}
	}

	return hosts, nil
}

type StoreHTTPLogEntryParams struct {
	Request  *http.Request
	Response *http.Response
}

func (srv *service) StoreHTTPLogEntry(ctx context.Context, params StoreHTTPLogEntryParams) error {
	hostname := params.Request.Host
	host, err := srv.findHostByHostname(ctx, hostname)
	if err != nil {
		return fmt.Errorf("hosts: failed to find host by hostname %q: %w", hostname, err)
	}

	now := time.Now().UTC()
	id := ulid.MustNew(ulid.Timestamp(now), ulidEntropy)

	entry := HTTPLogEntry{
		ID:       id,
		HostID:   host.ID,
		Request:  params.Request,
		Response: params.Response,
	}

	err = srv.database.StoreHTTPLogEntry(ctx, entry)
	if err != nil {
		return fmt.Errorf("hosts: failed to store HTTP log entry: %w", err)
	}

	srv.logger.Info("Stored HTTP log entry.",
		zap.String("id", entry.ID.String()),
		zap.String("hostId", entry.HostID.String()),
		zap.String("host", params.Request.Host),
		zap.String("url", params.Request.URL.String()),
		zap.String("method", params.Request.Method),
	)

	return nil
}

func (srv *service) findHostByHostname(ctx context.Context, hostname string) (Host, error) {
	host, err := srv.database.FindHostByHostname(ctx, hostname)
	if err != nil {
		return Host{}, err
	}

	return host, nil
}
