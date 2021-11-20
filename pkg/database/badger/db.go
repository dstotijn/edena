package badger

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net/http/httputil"

	"github.com/dgraph-io/badger/v3"
	"github.com/dstotijn/edena/pkg/hosts"
	"github.com/oklog/ulid"
)

const (
	hostKeyPrefix     byte = 0x00
	hostHostnameIndex byte = 0x01

	httpLogKeyPrefix   byte = 0x10
	httpLogHostIDIndex byte = 0x11

	indexKeyMask byte = 0x0F // Secondary index keys use the last 4 bits
)

type Database struct {
	badger *badger.DB
}

func OpenDatabase(opts badger.Options) (*Database, error) {
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("badger: failed to open database: %w", err)
	}

	return &Database{badger: db}, nil
}

func (db *Database) Close() error {
	return db.badger.Close()
}

func (db *Database) StoreHosts(ctx context.Context, hosts ...hosts.Host) error {
	if len(hosts) == 0 {
		return errors.New("badger: hosts cannot be 0 length")
	}

	entries := make([]*badger.Entry, 0, len(hosts)*2)
	for _, host := range hosts {
		buf := bytes.Buffer{}
		err := gob.NewEncoder(&buf).Encode(host)
		if err != nil {
			return fmt.Errorf("badger: failed to encode host: %w", err)
		}
		entries = append(entries,
			// Host itself
			&badger.Entry{
				Key:   entryKey(hostKeyPrefix, 0, host.ID[:]),
				Value: buf.Bytes(),
			},
			// Hostname index
			&badger.Entry{
				Key: entryKey(hostKeyPrefix, hostHostnameIndex, append([]byte(host.Hostname+"#"), host.ID[:]...)),
			},
		)
	}

	err := db.badger.Update(func(txn *badger.Txn) error {
		for i := range entries {
			err := txn.SetEntry(entries[i])
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("badger: failed to commit transaction: %w", err)
	}

	return nil
}

func (db *Database) FindHostByID(ctx context.Context, hostID ulid.ULID) (hosts.Host, error) {
	var rawHost []byte

	err := db.badger.View(func(txn *badger.Txn) error {
		item, err := txn.Get(entryKey(hostKeyPrefix, 0, hostID[:]))
		if err != nil {
			return err
		}

		rawHost, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return nil
	})
	if err == badger.ErrKeyNotFound {
		return hosts.Host{}, hosts.ErrHostNotFound
	}
	if err != nil {
		return hosts.Host{}, fmt.Errorf("badger: failed to commit transaction: %w", err)
	}

	host := hosts.Host{}
	err = gob.NewDecoder(bytes.NewReader(rawHost)).Decode(&host)
	if err != nil {
		return hosts.Host{}, fmt.Errorf("badger: failed to decode host: %w", err)
	}

	return host, nil
}

func (db *Database) FindHostByHostname(ctx context.Context, hostname string) (hosts.Host, error) {
	var rawHost []byte

	err := db.badger.View(func(txn *badger.Txn) error {
		var hostIndexKey []byte

		prefix := entryKey(hostKeyPrefix, hostHostnameIndex, []byte(hostname+"#"))
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			hostIndexKey = it.Item().KeyCopy(hostIndexKey)
			break
		}
		if hostIndexKey == nil {
			return hosts.ErrHostNotFound
		}

		// The host ID is the part of the index item key *after* the `#`.
		hostID := hostIndexKey[bytes.Index(hostIndexKey, []byte("#"))+1:]

		item, err := txn.Get(entryKey(hostKeyPrefix, 0, hostID))
		if err != nil {
			return err
		}

		rawHost, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return nil
	})
	if err == hosts.ErrHostNotFound || err == badger.ErrKeyNotFound {
		return hosts.Host{}, hosts.ErrHostNotFound
	}
	if err != nil {
		return hosts.Host{}, fmt.Errorf("badger: failed to commit transaction: %w", err)
	}

	host := hosts.Host{}
	err = gob.NewDecoder(bytes.NewReader(rawHost)).Decode(&host)
	if err != nil {
		return hosts.Host{}, fmt.Errorf("badger: failed to decode host: %w", err)
	}

	return host, nil
}

type httpLogEntry struct {
	ID          ulid.ULID
	HostID      ulid.ULID
	RawRequest  []byte
	RawResponse []byte
}

func (db *Database) StoreHTTPLogEntry(ctx context.Context, entry hosts.HTTPLogEntry) error {
	rawReq, err := httputil.DumpRequest(entry.Request, true)
	if err != nil {
		return fmt.Errorf("badger: failed to dump HTTP request: %w", err)
	}

	rawRes, err := httputil.DumpResponse(entry.Response, true)
	if err != nil {
		return fmt.Errorf("badger: failed to dump HTTP request: %w", err)
	}

	buf := bytes.Buffer{}
	err = gob.NewEncoder(&buf).Encode(httpLogEntry{
		ID:          entry.ID,
		HostID:      entry.HostID,
		RawRequest:  rawReq,
		RawResponse: rawRes,
	})
	if err != nil {
		return fmt.Errorf("badger: failed to encode log entry: %w", err)
	}

	entries := []*badger.Entry{
		// HTTP log itself
		{
			Key:   entryKey(httpLogKeyPrefix, 0, entry.ID[:]),
			Value: buf.Bytes(),
		},
		// Index by host ID
		{
			Key: entryKey(httpLogKeyPrefix, httpLogHostIDIndex, append(entry.HostID[:], entry.ID[:]...)),
		},
	}

	err = db.badger.Update(func(txn *badger.Txn) error {
		for i := range entries {
			err := txn.SetEntry(entries[i])
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("badger: failed to commit transaction: %w", err)
	}

	return nil
}

func (db *Database) ListHTTPLogEntries(ctx context.Context, params hosts.ListHTTPLogEntriesParams) ([]hosts.HTTPLogEntry, error) {
	var httpLogEntries []hosts.HTTPLogEntry

	err := db.badger.View(func(txn *badger.Txn) error {
		var rawHTTPLogEntry []byte
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for _, hostID := range params.HostIDs {
			var hostIndexKey []byte
			prefix := entryKey(httpLogKeyPrefix, httpLogHostIDIndex, hostID[:])

			it.Rewind()

			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				hostIndexKey = it.Item().KeyCopy(hostIndexKey)

				// The HTTP log entry ID starts *after* the first index byte
				// and the 16 byte host ID.
				httpLogEntryID := hostIndexKey[17:]

				item, err := txn.Get(entryKey(httpLogKeyPrefix, 0, httpLogEntryID))
				if err != nil {
					return err
				}

				rawHTTPLogEntry, err = item.ValueCopy(rawHTTPLogEntry)
				if err != nil {
					return err
				}

				httpLogEntry := hosts.HTTPLogEntry{}
				err = gob.NewDecoder(bytes.NewReader(rawHTTPLogEntry)).Decode(&httpLogEntry)
				if err != nil {
					return err
				}

				httpLogEntries = append(httpLogEntries, httpLogEntry)
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("badger: failed to commit transaction: %w", err)
	}

	return httpLogEntries, nil
}

func entryKey(prefix, indexKey byte, indexValue []byte) []byte {
	key := make([]byte, 1+len(indexValue))
	// Key consists of: <4 bits for prefix><4 bits for index identifier><value>
	key[0] = (indexKey & indexKeyMask) | prefix
	copy(key[1:len(indexValue)+1], indexValue)

	return key
}
