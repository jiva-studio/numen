// Package tagcache keeps the tags of the notes of each vault: read from the frontmatter the host holds
// when the plugin has none, brought up to date from the file change feed of the vault, and written to
// the plugin's folder of the vault's store with the cursor of the feed.
package tagcache

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tagfile"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
)

// FileName is the file of the plugin's store folder the tags of a vault are written to.
const FileName = "tags.json"

// CursorFile is the file of the plugin's store folder the cursor of the feed of a vault is kept in.
const CursorFile = "tags.cursor.json"

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

// Properties writes the frontmatter of the notes of a vault.
type Properties interface {
	WriteFrontmatter(ctx context.Context, writes []host.FrontmatterWrite) ([]host.FrontmatterResult, error)
}

// Vault is what the cache of one vault reads and writes.
type Vault struct {
	Frontmatter Frontmatter
	Feed        host.FileChangeLister
	Store       Store
	Properties  Properties
}

// Caches holds the tags of each vault the plugin was asked about. Open gives the vault of an id.
type Caches struct {
	open   func(id domain.VaultID) Vault
	mu     sync.Mutex
	vaults map[domain.VaultID]*vaultTags
}

// vaultTags is the tags of the notes of one vault that have any, by the number the index keeps each
// note under. Updates and answers of one vault take mu in turn. isLoaded is false until tags.json was
// read, and again after an update failed.
type vaultTags struct {
	mu       sync.Mutex
	isLoaded bool
	notes    map[int64]tagfile.Note
}

func New(open func(id domain.VaultID) Vault) *Caches {
	return &Caches{open: open, mu: sync.Mutex{}, vaults: map[domain.VaultID]*vaultTags{}}
}

func (c *Caches) getVault(id domain.VaultID) *vaultTags {
	c.mu.Lock()
	defer c.mu.Unlock()
	one, ok := c.vaults[id]
	if !ok {
		one = &vaultTags{mu: sync.Mutex{}, isLoaded: false, notes: map[int64]tagfile.Note{}}
		c.vaults[id] = one
	}
	return one
}

// GetMap returns the tags of the notes of the vault that have any, by path, after bringing the tags up
// to date.
func (c *Caches) GetMap(ctx context.Context, id domain.VaultID) (map[string][]string, error) {
	one := c.getVault(id)
	one.mu.Lock()
	defer one.mu.Unlock()
	if err := c.update(ctx, id, one); err != nil {
		return nil, err
	}
	return one.getTagsByPath(), nil
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
	return tags.GetTree(one.getTagsByPath(), under), nil
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

func (v *vaultTags) getTagsByPath() map[string][]string {
	byPath := make(map[string][]string, len(v.notes))
	for _, note := range v.notes {
		byPath[note.Path] = slices.Clone(note.Tags)
	}
	return byPath
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
		notes = map[int64]tagfile.Note{}
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

func readTags(ctx context.Context, frontmatter Frontmatter, paths []string) (map[int64]tagfile.Note, error) {
	notes := map[int64]tagfile.Note{}
	query := host.FrontmatterQuery{Paths: paths, Keys: []string{tags.FrontmatterKey}}
	for {
		page, err := frontmatter.ReadFrontmatter(ctx, query)
		if err != nil {
			return nil, err
		}
		for _, note := range page.Notes {
			if found := tags.Parse(note.Values[tags.FrontmatterKey]); len(found) > 0 {
				notes[note.SourceID] = tagfile.Note{Path: note.Path, Tags: found}
			}
		}
		if page.NextPageToken == "" {
			return notes, nil
		}
		query.PageToken = page.NextPageToken
	}
}

func readFile(ctx context.Context, store Store) (map[int64]tagfile.Note, error) {
	content, err := store.Read(ctx, FileName)
	if errors.Is(err, host.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", FileName, err)
	}
	return parseFile(content), nil
}

func parseFile(content []byte) map[int64]tagfile.Note {
	return tagfile.Parse(content)
}

func write(ctx context.Context, store Store, notes map[int64]tagfile.Note) error {
	content, err := tagfile.Encode(maps.Clone(notes))
	if err != nil {
		return err
	}
	if err := store.Write(ctx, FileName, content); err != nil {
		return fmt.Errorf("write %s: %w", FileName, err)
	}
	return nil
}

// ListTaggedPaths returns, in path order, the notes of the vault with a tag that is under or is the
// parent; an empty parent lists every note with a tag. When the tags cannot be brought up to date, it
// answers from the tags it holds and isStale is true.
func (c *Caches) ListTaggedPaths(ctx context.Context, id domain.VaultID, parent string) (paths []string, isStale bool, err error) {
	return c.listPaths(ctx, id, func(tag string) bool { return tags.IsUnder(tag, parent) })
}

// ListExactlyTaggedPaths returns, in path order, the notes of the vault with the tag itself, compared
// as tags.GetKey does; a note with only a tag nested under it is left out. Staleness is as for
// ListTaggedPaths.
func (c *Caches) ListExactlyTaggedPaths(ctx context.Context, id domain.VaultID, tag string) (paths []string, isStale bool, err error) {
	key := tags.GetKey(tag)
	return c.listPaths(ctx, id, func(have string) bool { return tags.GetKey(have) == key })
}

func (c *Caches) listPaths(ctx context.Context, id domain.VaultID, isMatch func(tag string) bool) (paths []string, isStale bool, err error) {
	one := c.getVault(id)
	one.mu.Lock()
	defer one.mu.Unlock()
	if err := c.update(ctx, id, one); err != nil {
		if ctx.Err() != nil || len(one.notes) == 0 {
			return nil, false, err
		}
		isStale = true
	}
	for _, note := range one.notes {
		if slices.ContainsFunc(note.Tags, isMatch) {
			paths = append(paths, note.Path)
		}
	}
	slices.Sort(paths)
	return paths, isStale, nil
}
