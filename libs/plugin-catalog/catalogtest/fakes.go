// Package catalogtest holds the test doubles of the catalog ports and the contract every
// catalog.Source satisfies.
package catalogtest

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
)

// FakeSource is a source over a fixed list of entries, or one that fails.
type FakeSource struct {
	name    string
	entries []catalog.Entry
	// Err is returned by every call. The entries are returned with it, as a source that read part of
	// its place does.
	Err error
}

// NewFakeSource returns a source named name that lists the entries, each carrying the name.
func NewFakeSource(name string, entries ...catalog.Entry) *FakeSource {
	copies := slices.Clone(entries)
	for i := range copies {
		copies[i].Source = name
	}
	return &FakeSource{name: name, entries: copies, Err: nil}
}

// Name is the name given to NewFakeSource.
func (f *FakeSource) Name() string { return f.name }

// List returns a copy of the entries and Err.
func (f *FakeSource) List(ctx context.Context) ([]catalog.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return slices.Clone(f.entries), f.Err
}

// Get returns the entry of the plugin and Err.
func (f *FakeSource) Get(ctx context.Context, id string) (catalog.Entry, error) {
	if err := ctx.Err(); err != nil {
		return catalog.Entry{}, err
	}
	for _, entry := range f.entries {
		if entry.ID == id {
			return entry, f.Err
		}
	}
	return catalog.Entry{}, errors.Join(fmt.Errorf("plugin %q: %w", id, catalog.ErrNotFound), f.Err)
}

// FakeCache is a cache in memory.
type FakeCache struct {
	mu       sync.Mutex
	snapshot *catalog.Snapshot
	// WriteErr is returned by Write, which then stores nothing.
	WriteErr error
	// Writes counts the calls to Write.
	Writes int
}

// Read returns the snapshot, or catalog.ErrEmptyCache.
func (f *FakeCache) Read(ctx context.Context) (catalog.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return catalog.Snapshot{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.snapshot == nil {
		return catalog.Snapshot{}, catalog.ErrEmptyCache
	}
	return *f.snapshot, nil
}

// Write keeps the snapshot.
func (f *FakeCache) Write(ctx context.Context, snapshot catalog.Snapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Writes++
	if f.WriteErr != nil {
		return f.WriteErr
	}
	f.snapshot = &catalog.Snapshot{Entries: slices.Clone(snapshot.Entries), At: snapshot.At}
	return nil
}
