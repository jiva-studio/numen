// Package snapshot keeps the snapshot of the index in the cache directory the host grants the
// plugin, which the host removes when the plugin is updated or uninstalled.
package snapshot

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
)

const (
	name = "index"
	key  = "snapshot"
)

var errNone = errors.New("snapshot: none kept")

// Store is the snapshot of the index in one file of the cache directory.
type Store struct {
	cache *cache.Cache[ports.Snapshot]
}

// New returns a store that writes to the directory dir, or to the cache directory of the plugin
// when dir is empty.
func New(dir string) *Store {
	options := []cache.Option{cache.WithDisk(true)}
	if dir != "" {
		options = append(options, cache.WithDir(dir))
	}
	return &Store{cache: cache.New[ports.Snapshot](name, options...)}
}

// Load returns the kept snapshot; a missing or unreadable file is no snapshot.
func (s *Store) Load(ctx context.Context) (ports.Snapshot, bool) {
	value, isFound, err := s.cache.Get(ctx, key, func(context.Context) (ports.Snapshot, error) {
		return ports.Snapshot{}, errNone
	})
	if err != nil || !isFound {
		return ports.Snapshot{}, false
	}
	return value, true
}

// Save replaces the kept snapshot.
func (s *Store) Save(snapshot ports.Snapshot) error {
	s.cache.Put(key, snapshot)
	return nil
}

type cursorStore struct {
	get func() host.FileChangeCursor
	set func(host.FileChangeCursor)
}

// NewCursorStore returns the store of the feed cursor over a cursor the index holds.
func NewCursorStore(get func() host.FileChangeCursor, set func(host.FileChangeCursor)) changes.Store {
	return cursorStore{get: get, set: set}
}

func (c cursorStore) Read(_ context.Context, _ string) ([]byte, error) {
	cursor := c.get()
	if cursor == (host.FileChangeCursor{}) {
		return nil, host.ErrNotExist
	}
	return json.Marshal(cursor)
}

func (c cursorStore) Write(_ context.Context, _ string, content []byte) error {
	var cursor host.FileChangeCursor
	if err := json.Unmarshal(content, &cursor); err != nil {
		return err
	}
	c.set(cursor)
	return nil
}
