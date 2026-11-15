// Package tagcache keeps the tags of the notes of each vault: read from the frontmatter the host holds
// when the plugin has none, brought up to date from the file change feed of the vault, and written to
// the plugin's folder of the vault's store with the cursor of the feed.
package tagcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
)

// FileName is the file of the plugin's store folder the tags of a vault are written to.
const FileName = "tags.json"

// CursorFile is the file of the plugin's store folder the cursor of the feed of a vault is kept in.
const CursorFile = "tags.cursor.json"

const fileVersion = 2

// readPathLimit is the most notes one read of frontmatter names.
const readPathLimit = 1000

// Frontmatter reads the frontmatter of the notes of a vault.
type Frontmatter interface {
	ReadFrontmatter(ctx context.Context, q host.FrontmatterQuery) (host.FrontmatterPage, error)
}

// Store reads and writes a file of the plugin's folder of a vault's store.
type Store interface {
	Read(ctx context.Context, name string) ([]byte, error)
	Write(ctx context.Context, name string, content []byte) error
}

// Vault is what the cache of one vault reads and writes.
type Vault struct {
	Frontmatter Frontmatter
	Feed        host.FileChangeLister
	Store       Store
}

// Caches holds the tags of each vault the plugin was asked about. Open gives the vault of an id.
type Caches struct {
	open   func(id domain.VaultID) Vault
	mu     sync.Mutex
	vaults map[domain.VaultID]*vaultTags
}

// Note is the tags of one note and where it is now.
type Note struct {
	Path string   `json:"path"`
	Tags []string `json:"tags"`
}

// vaultTags is the tags of the notes of one vault that have any, by the number the index keeps each
// note under. Updates and answers of one vault take mu in turn. isLoaded is false until tags.json was
// read, and again after an update failed.
type vaultTags struct {
	mu       sync.Mutex
	isLoaded bool
	notes    map[int64]Note
}

func New(open func(id domain.VaultID) Vault) *Caches {
	return &Caches{open: open, mu: sync.Mutex{}, vaults: map[domain.VaultID]*vaultTags{}}
}

func (c *Caches) getVault(id domain.VaultID) *vaultTags {
	c.mu.Lock()
	defer c.mu.Unlock()
	one, ok := c.vaults[id]
	if !ok {
		one = &vaultTags{mu: sync.Mutex{}, isLoaded: false, notes: map[int64]Note{}}
		c.vaults[id] = one
	}
	return one
}

// GetTree returns the tree of the tags of the vault, as tags.GetTree builds it, after bringing the
// tags up to date.
func (c *Caches) GetTree(ctx context.Context, id domain.VaultID, under string) ([]tags.Node, error) {
	one := c.getVault(id)
	one.mu.Lock()
	defer one.mu.Unlock()
	if err := c.update(ctx, id, one); err != nil {
		return nil, err
	}
	byPath := make(map[string][]string, len(one.notes))
	for _, note := range one.notes {
		byPath[note.Path] = note.Tags
	}
	return tags.GetTree(byPath, under), nil
}

// UpdateVault brings the tags of the vault up to date with its file change feed.
func (c *Caches) UpdateVault(ctx context.Context, id domain.VaultID) error {
	one := c.getVault(id)
	one.mu.Lock()
	defer one.mu.Unlock()
	return c.update(ctx, id, one)
}

func (c *Caches) update(ctx context.Context, id domain.VaultID, one *vaultTags) error {
	vault := c.open(id)
	from := changes.Source{Feed: vault.Feed, Store: vault.Store, CursorFile: CursorFile}
	if err := one.load(ctx, vault, from); err != nil {
		return err
	}
	events := []string{host.EventNoteFrontmatterChanged, host.EventFileRemoved, host.EventFileMoved}
	err := changes.Follow(ctx, from, events,
		func(ctx context.Context, batch []host.FileChange) error { return one.apply(ctx, vault, batch) },
		func(ctx context.Context) error { return one.review(ctx, vault) })
	if err != nil {
		one.isLoaded = false
		return fmt.Errorf("update the tags: %w", err)
	}
	return nil
}

func (v *vaultTags) load(ctx context.Context, vault Vault, from changes.Source) error {
	if v.isLoaded {
		return nil
	}
	notes, err := readFile(ctx, vault.Store)
	if err != nil {
		return err
	}
	if notes == nil {
		if err := changes.Forget(ctx, from); err != nil {
			return err
		}
		notes = map[int64]Note{}
	}
	v.notes, v.isLoaded = notes, true
	return nil
}

func (v *vaultTags) apply(ctx context.Context, vault Vault, batch []host.FileChange) error {
	var reread []host.FileChange
	for _, change := range batch {
		switch change.Event {
		case host.EventFileRemoved:
			delete(v.notes, change.SourceID)
		case host.EventFileMoved:
			if note, ok := v.notes[change.SourceID]; ok {
				note.Path = change.Path
				v.notes[change.SourceID] = note
			}
		case host.EventNoteFrontmatterChanged:
			reread = append(reread, change)
		}
	}
	for part := range slices.Chunk(reread, readPathLimit) {
		paths := make([]string, 0, len(part))
		for _, change := range part {
			paths = append(paths, change.Path)
		}
		found, err := readTags(ctx, vault.Frontmatter, paths)
		if err != nil {
			return fmt.Errorf("read the tags of %d notes: %w", len(paths), err)
		}
		for _, change := range part {
			delete(v.notes, change.SourceID)
		}
		maps.Copy(v.notes, found)
	}
	return write(ctx, vault.Store, v.notes)
}

func (v *vaultTags) review(ctx context.Context, vault Vault) error {
	notes, err := readTags(ctx, vault.Frontmatter, nil)
	if err != nil {
		return fmt.Errorf("read the tags of the notes: %w", err)
	}
	v.notes = notes
	return write(ctx, vault.Store, v.notes)
}

func readTags(ctx context.Context, frontmatter Frontmatter, paths []string) (map[int64]Note, error) {
	notes := map[int64]Note{}
	query := host.FrontmatterQuery{Paths: paths, Keys: []string{tags.FrontmatterKey}}
	for {
		page, err := frontmatter.ReadFrontmatter(ctx, query)
		if err != nil {
			return nil, err
		}
		for _, note := range page.Notes {
			if found := tags.Parse(note.Values[tags.FrontmatterKey]); len(found) > 0 {
				notes[note.SourceID] = Note{Path: note.Path, Tags: found}
			}
		}
		if page.NextPageToken == "" {
			return notes, nil
		}
		query.PageToken = page.NextPageToken
	}
}

// file is the content of tags.json: the tags of each note that has any, by the number the index keeps
// the note under. A missing tags.json, or one of another version, holds no tags, and the next update
// reads every note.
type file struct {
	Version int            `json:"version"`
	Notes   map[int64]Note `json:"notes"`
}

func readFile(ctx context.Context, store Store) (map[int64]Note, error) {
	content, err := store.Read(ctx, FileName)
	if errors.Is(err, host.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", FileName, err)
	}
	return parseFile(content), nil
}

func parseFile(content []byte) map[int64]Note {
	var stored file
	if json.Unmarshal(content, &stored) != nil || stored.Version != fileVersion {
		return nil
	}
	return stored.Notes
}

func write(ctx context.Context, store Store, notes map[int64]Note) error {
	content, err := json.Marshal(file{Version: fileVersion, Notes: maps.Clone(notes)})
	if err != nil {
		return err
	}
	if err := store.Write(ctx, FileName, content); err != nil {
		return fmt.Errorf("write %s: %w", FileName, err)
	}
	return nil
}
