package hosts

import (
	"context"
	"math/rand"
	"time"

	"go.uber.org/zap"
)

var ulidEntropy = rand.New(rand.NewSource(time.Now().UnixNano()))

type Service interface {
	CreateHosts(ctx context.Context, amount int) ([]Host, error)
	StoreHTTPLogEntry(ctx context.Context, params StoreHTTPLogEntryParams) error
}

type service struct {
	baseHostname string
	database     Database
	logger       *zap.Logger
}

type serviceOption func(*service)

type Database interface {
	StoreHosts(ctx context.Context, hosts ...Host) error
	StoreHTTPLogEntry(ctx context.Context, entry HTTPLogEntry) error
	FindHostByHostname(ctx context.Context, hostname string) (Host, error)
}

func NewService(opts ...serviceOption) Service {
	srv := &service{}

	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

// WithBaseHostname provides a base hostname, to use when generating hostnames.
func WithBaseHostname(baseHostname string) serviceOption {
	return func(srv *service) {
		srv.baseHostname = baseHostname
	}
}

// WithDatabase provides a database, which is used for storing hosts data.
func WithDatabase(db Database) serviceOption {
	return func(srv *service) {
		srv.database = db
	}
}

// WithLogger provides a logger, which is used for logging hosts management
// events.
func WithLogger(logger *zap.Logger) serviceOption {
	return func(srv *service) {
		srv.logger = logger
	}
}
