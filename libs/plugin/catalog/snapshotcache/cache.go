// Package snapshotcache keeps the last snapshot of a catalog source in a JSON file.
package snapshotcache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/libs/atomicfile"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Cache is the file a snapshot is written to.
type Cache struct {
	path string
}

var errNoPath = errors.New("the snapshot cache names no file")

// New returns the cache in the file, which is created on the first Write.
func New(path string) (*Cache, error) {
	if path == "" {
		return nil, errNoPath
	}
	return &Cache{path: path}, nil
}

type snapshotFile struct {
	At      time.Time   `json:"at"`
	Entries []entryFile `json:"entries"`
}

type entryFile struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Description string        `json:"description"`
	Category    string        `json:"category"`
	TabKinds    []tabKindFile `json:"tabKinds"`
	Feature     string        `json:"feature"`
	Source      string        `json:"source"`
	State       catalog.State `json:"state"`
	Targets     []string      `json:"targets"`
	Protocol    int           `json:"protocol"`
}

type tabKindFile struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
}

// Read returns the snapshot, or catalog.ErrEmptyCache when the file does not exist.
func (c *Cache) Read(ctx context.Context) (catalog.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return catalog.Snapshot{}, err
	}
	data, err := os.ReadFile(c.path)
	if errors.Is(err, fs.ErrNotExist) {
		return catalog.Snapshot{}, catalog.ErrEmptyCache
	}
	if err != nil {
		return catalog.Snapshot{}, fmt.Errorf("read the snapshot: %w", err)
	}
	var stored snapshotFile
	if err := json.Unmarshal(data, &stored); err != nil {
		return catalog.Snapshot{}, fmt.Errorf("decode the snapshot: %w", err)
	}
	entries := make([]catalog.Entry, 0, len(stored.Entries))
	for _, one := range stored.Entries {
		entries = append(entries, newEntry(one))
	}
	return catalog.Snapshot{Entries: entries, At: stored.At}, nil
}

// Write replaces the file with the snapshot. The file is renamed into place, so a reader sees the
// old snapshot or the new one.
func (c *Cache) Write(ctx context.Context, snapshot catalog.Snapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	stored := snapshotFile{At: snapshot.At, Entries: make([]entryFile, 0, len(snapshot.Entries))}
	for _, entry := range snapshot.Entries {
		stored.Entries = append(stored.Entries, newEntryFile(entry))
	}
	data, err := json.Marshal(stored)
	// gomutants:disable-next-line BRANCH_IF reason="snapshotFile contains only JSON-serializable primitives"
	if err != nil {
		return fmt.Errorf("encode the snapshot: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return fmt.Errorf("create the snapshot folder: %w", err)
	}
	if err := atomicfile.WriteFile(c.path, bytes.NewReader(data), 0o600); err != nil {
		return fmt.Errorf("write the snapshot: %w", err)
	}
	return nil
}

func newEntryFile(entry catalog.Entry) entryFile {
	kinds := make([]tabKindFile, 0, len(entry.Contributes.TabKinds))
	for _, kind := range entry.Contributes.TabKinds {
		kinds = append(kinds, tabKindFile{Kind: kind.Kind, Title: kind.Title, Icon: kind.Icon})
	}
	return entryFile{
		ID:          entry.ID,
		Name:        entry.Name,
		Version:     entry.Version,
		Description: entry.Description,
		Category:    entry.Category,
		TabKinds:    kinds,
		Feature:     entry.Contributes.Feature,
		Source:      entry.Source,
		State:       entry.State,
		Targets:     entry.Requirements.Targets,
		Protocol:    entry.Requirements.Protocol,
	}
}

func newEntry(stored entryFile) catalog.Entry {
	var kinds []manifest.TabKind
	for _, kind := range stored.TabKinds {
		kinds = append(kinds, manifest.TabKind{Kind: kind.Kind, Title: kind.Title, Icon: kind.Icon, When: "", Rank: domain.RankDefault, IsNoteEditor: false})
	}
	return catalog.Entry{
		ID:              stored.ID,
		Name:            stored.Name,
		Version:         stored.Version,
		Description:     stored.Description,
		Category:        stored.Category,
		Contributes:     manifest.Contributes{TabKinds: kinds, Feature: stored.Feature, Category: stored.Category},
		Source:          stored.Source,
		State:           stored.State,
		Requirements:    manifest.Requirements{Targets: stored.Targets, Protocol: stored.Protocol},
		Incompatibility: nil,
		Warnings:        nil,
	}
}
