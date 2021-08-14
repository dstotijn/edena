package http

import (
	"net/http"

	"github.com/oklog/ulid"
)

type LogEntry struct {
	ID          ulid.ULID
	HostID      ulid.ULID
	Request     http.Request
	Response    http.Response
	RawRequest  []byte
	RawResponse []byte
}
