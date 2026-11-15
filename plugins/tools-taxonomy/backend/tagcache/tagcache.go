// Package tagcache keeps the tags of the notes of each vault: read from the frontmatter the host
// holds at the first use of a vault, read again for the notes an update names, and written to the
// plugin's folder of the vault's store.
package tagcache

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/tags"
)

// FileName is the file of the plugin's store folder the tags of a vault are written to.
const FileName = "tags.json"

const fileVersion = 1

// Frontmatter reads the frontmatter of the notes of a vault.
type Frontmatter interface {
	ReadFrontmatter(ctx context.Context, q host.FrontmatterQuery) (host.FrontmatterPage, error)
}

// Store writes a file of the plugin's folder of a vault's store.
type Store interface {
	Write(ctx context.Context, name string, content []byte) error
}

// Vault is what the cache of one vault reads and writes.
type Vault struct {
	Frontmatter Frontmatter
	Store       Store
}

// Caches holds the tags of each vault the plugin was asked about. Open gives the vault of an id.
type Caches struct {
	open   func(id domain.VaultID) Vault
	mu     sync.Mutex
	vaults map[domain.VaultID]*vaultTags
}

// vaultTags is the tags of the notes of one vault. Updates and answers of one vault take mu in turn.
// isRead is false until every note was read, and again after an update failed.
type vaultTags struct {
	mu     sync.Mutex
	isRead bool
	notes  map[string][]string
}

func New(open func(id domain.VaultID) Vault) *Caches {
	return &Caches{open: open, mu: sync.Mutex{}, vaults: map[domain.VaultID]*vaultTags{}}
}

func (c *Caches) getVault(id domain.VaultID) *vaultTags {
	c.mu.Lock()
	defer c.mu.Unlock()
	one, ok := c.vaults[id]
	if !ok {
		one = &vaultTags{mu: sync.Mutex{}, isRead: false, notes: map[string][]string{}}
		c.vaults[id] = one
	}
	return one
}

// GetTree returns the tree of the tags of the vault, as tags.GetTree builds it. The first call of a
// vault reads the tags of every note.
func (c *Caches) GetTree(ctx context.Context, id domain.VaultID, under string) ([]tags.Node, error) {
	one := c.getVault(id)
	one.mu.Lock()
	defer one.mu.Unlock()
	if err := c.readAll(ctx, id, one); err != nil {
		return nil, err
	}
	return tags.GetTree(one.notes, under), nil
}

// UpdateNotes reads the tags of the notes at the paths again in one read, and drops each note the vault
// has none at. A vault not read yet is read whole instead. A failed update leaves the vault to be read
// whole at its next use.
func (c *Caches) UpdateNotes(ctx context.Context, id domain.VaultID, paths []string) error {
	one := c.getVault(id)
	one.mu.Lock()
	defer one.mu.Unlock()
	if !one.isRead {
		return c.readAll(ctx, id, one)
	}
	vault := c.open(id)
	found, err := readTags(ctx, vault.Frontmatter, paths)
	if err != nil {
		one.isRead = false
		return fmt.Errorf("read the tags of %d notes: %w", len(paths), err)
	}
	if !one.replace(paths, found) {
		return nil
	}
	if err := write(ctx, vault.Store, one.notes); err != nil {
		one.isRead = false
		return err
	}
	return nil
}

func (v *vaultTags) replace(paths []string, found map[string][]string) bool {
	isChanged := false
	for _, path := range paths {
		before, had := v.notes[path]
		after, has := found[path]
		if had == has && slices.Equal(before, after) {
			continue
		}
		isChanged = true
		if has {
			v.notes[path] = after
		} else {
			delete(v.notes, path)
		}
	}
	return isChanged
}

func (c *Caches) readAll(ctx context.Context, id domain.VaultID, one *vaultTags) error {
	if one.isRead {
		return nil
	}
	vault := c.open(id)
	notes, err := readTags(ctx, vault.Frontmatter, nil)
	if err != nil {
		return fmt.Errorf("read the tags of the notes: %w", err)
	}
	if err := write(ctx, vault.Store, notes); err != nil {
		return err
	}
	one.notes, one.isRead = notes, true
	return nil
}

func readTags(ctx context.Context, frontmatter Frontmatter, paths []string) (map[string][]string, error) {
	notes := map[string][]string{}
	query := host.FrontmatterQuery{Paths: paths, Keys: []string{tags.FrontmatterKey}}
	for {
		page, err := frontmatter.ReadFrontmatter(ctx, query)
		if err != nil {
			return nil, err
		}
		for _, note := range page.Notes {
			if found := tags.Parse(note.Values[tags.FrontmatterKey]); len(found) > 0 {
				notes[note.Path] = found
			}
		}
		if page.NextPageToken == "" {
			return notes, nil
		}
		query.PageToken = page.NextPageToken
	}
}

// file is the content of tags.json: the tags of each note that has any, by path.
type file struct {
	Version int                 `json:"version"`
	Notes   map[string][]string `json:"notes"`
}

func write(ctx context.Context, store Store, notes map[string][]string) error {
	content, err := json.Marshal(file{Version: fileVersion, Notes: maps.Clone(notes)})
	if err != nil {
		return err
	}
	if err := store.Write(ctx, FileName, content); err != nil {
		return fmt.Errorf("write %s: %w", FileName, err)
	}
	return nil
}
