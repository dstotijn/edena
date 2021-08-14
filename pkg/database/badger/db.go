package badger

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"

	"github.com/dgraph-io/badger/v3"

	"github.com/dstotijn/edena/pkg/http"
)

const (
	httpLogKeyPrefix   byte = 0x80 // HTTP log entries have first key bit set to 1
	httpLogHostIDIndex byte = 0x81 // Used to index HTTP logs by "host ID"
	indexKeyMask       byte = 0x0F // Secondary index keys use the last 4 bits
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

func (db *Database) StoreHTTPLogEntry(ctx context.Context, entry http.LogEntry) error {
	buf := bytes.Buffer{}
	err := gob.NewEncoder(&buf).Encode(entry)
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

func entryKey(prefix, indexKey byte, indexValue []byte) []byte {
	key := make([]byte, 1+len(indexValue))
	// Key consists of: <4 bits for prefix><4 bits for index identifier><value>
	key[0] = (indexKey & indexKeyMask) | prefix
	copy(key[1:len(indexValue)+1], indexValue)

	return key
}
