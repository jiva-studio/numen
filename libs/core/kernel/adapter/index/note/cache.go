package note

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// Cache holds read answers for indexed notes until a write commits to the index.
type Cache struct {
	mu         sync.RWMutex
	generation uint64
	vaults     map[domain.VaultID]*vaultCache
}

type vaultCache struct {
	row            int64
	hasRow         bool
	isGraphLoaded  bool
	linksOf        map[string][]domain.ResolvedLink
	backlinksOf    map[string][]domain.ResolvedLink
	named          map[string][]string
	notes          map[string]cachedNoteRef
	types          map[string]cachedNoteType
	heads          map[string]cachedHead
	rawFrontmatter map[string]sql.NullString
}

type cachedNoteRef struct {
	ref    domain.NoteRef
	exists bool
}

type cachedNoteType struct {
	noteType domain.NoteType
	exists   bool
}

type cachedHead struct {
	head   frontmatter.Head
	exists bool
}

type rawVaultLink struct {
	link   domain.ResolvedLink
	folded string
}

// NewCache creates an empty read cache.
func NewCache() *Cache {
	return &Cache{vaults: map[domain.VaultID]*vaultCache{}}
}

// Invalidate drops all cached entries.
func (c *Cache) Invalidate() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.generation++
	c.vaults = map[domain.VaultID]*vaultCache{}
	c.mu.Unlock()
}

func (c *Cache) getGeneration() uint64 {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.generation
}

func (c *Cache) isVaultGraphLoaded(vaultID domain.VaultID) bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.vaults[vaultID]
	return ok && v.isGraphLoaded
}

func (c *Cache) ensureGraphLoaded(ctx context.Context, db *sql.DB, vaultID domain.VaultID) error {
	if c == nil || c.isVaultGraphLoaded(vaultID) {
		return nil
	}
	gen := c.getGeneration()
	vault, ok := c.getVaultRow(vaultID)
	if !ok {
		var err error
		vault, err = vaultRow(ctx, db, vaultID)
		if errors.Is(err, errNoVault) {
			return nil
		}
		if err != nil {
			return err
		}
		c.setVaultRow(gen, vaultID, vault)
	}

	notes, types, rawFrontmatter, byPath, idByPath, byFolded, err := loadVaultNotes(ctx, db, vault)
	if err != nil {
		return err
	}
	order, grouped, noteIDs, err := loadVaultLinks(ctx, db, vault)
	if err != nil {
		return err
	}
	identified, err := loadIdentifiedNotes(ctx, db, noteIDs)
	if err != nil {
		return err
	}

	named := make(map[string][]string, len(byFolded))
	linksOf := make(map[string][]domain.ResolvedLink, len(byPath))
	backlinksOf := make(map[string][]domain.ResolvedLink, len(byPath))
	for _, from := range order {
		if err := ctx.Err(); err != nil {
			return err
		}
		raws := grouped[from]
		resolved := make([]domain.ResolvedLink, len(raws))
		for i, raw := range raws {
			l := raw.link
			resolveCachedLink(vaultID, from, &l, byPath, byFolded, named, identified)
			resolved[i] = l
			if l.ToVault == vaultID && l.To != "" && isBacklinkMatch(l, raw.folded, idByPath[l.To], byPath[l.To]) {
				backlinksOf[l.To] = append(backlinksOf[l.To], l)
			}
		}
		if deduped := dedupe(resolved); len(deduped) > 0 {
			linksOf[from] = deduped
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generation != gen {
		return nil
	}
	v := c.getOrCreateVault(vaultID)
	v.row = vault
	v.hasRow = true
	v.isGraphLoaded = true
	v.linksOf = linksOf
	v.backlinksOf = backlinksOf
	maps.Copy(v.named, named)
	v.notes = notes
	v.types = types
	v.rawFrontmatter = rawFrontmatter
	return nil
}

func (c *Cache) getOrCreateVault(vaultID domain.VaultID) *vaultCache {
	v, ok := c.vaults[vaultID]
	if !ok {
		v = &vaultCache{
			linksOf:     map[string][]domain.ResolvedLink{},
			backlinksOf: map[string][]domain.ResolvedLink{},
			named:       map[string][]string{},
			notes:       map[string]cachedNoteRef{},
			types:       map[string]cachedNoteType{},
			heads:       map[string]cachedHead{},
		}
		c.vaults[vaultID] = v
	}
	return v
}

func (c *Cache) getVaultRow(vaultID domain.VaultID) (int64, bool) {
	if c == nil {
		return 0, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.vaults[vaultID]
	if !ok || !v.hasRow {
		return 0, false
	}
	return v.row, true
}

func (c *Cache) setVaultRow(gen uint64, vaultID domain.VaultID, row int64) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generation != gen {
		return
	}
	v := c.getOrCreateVault(vaultID)
	v.row = row
	v.hasRow = true
}

func (c *Cache) splitLinksOf(
	vaultID domain.VaultID, paths []string,
) (uint64, map[string][]domain.ResolvedLink, []string) {
	if c == nil {
		return 0, map[string][]domain.ResolvedLink{}, paths
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	gen := c.generation
	v, ok := c.vaults[vaultID]
	if !ok {
		return gen, map[string][]domain.ResolvedLink{}, paths
	}
	cached := make(map[string][]domain.ResolvedLink, len(paths))
	var missing []string
	seen := map[string]bool{}
	for _, path := range paths {
		if links, found := v.linksOf[path]; found || v.isGraphLoaded {
			if len(links) > 0 {
				cached[path] = slices.Clone(links)
			}
			continue
		}
		if !seen[path] {
			seen[path] = true
			missing = append(missing, path)
		}
	}
	return gen, cached, missing
}

func (c *Cache) setLinksOf(
	gen uint64, vaultID domain.VaultID, queried []string, fetched map[string][]domain.ResolvedLink,
) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generation != gen {
		return
	}
	v := c.getOrCreateVault(vaultID)
	for _, path := range queried {
		v.linksOf[path] = slices.Clone(fetched[path])
	}
}

func (c *Cache) splitBacklinksOf(
	vaultID domain.VaultID, paths []string,
) (uint64, map[string][]domain.ResolvedLink, []string) {
	if c == nil {
		return 0, map[string][]domain.ResolvedLink{}, paths
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	gen := c.generation
	v, ok := c.vaults[vaultID]
	if !ok {
		return gen, map[string][]domain.ResolvedLink{}, paths
	}
	cached := make(map[string][]domain.ResolvedLink, len(paths))
	var missing []string
	seen := map[string]bool{}
	for _, path := range paths {
		if links, found := v.backlinksOf[path]; found || v.isGraphLoaded {
			if len(links) > 0 {
				cached[path] = slices.Clone(links)
			}
			continue
		}
		if !seen[path] {
			seen[path] = true
			missing = append(missing, path)
		}
	}
	return gen, cached, missing
}

func (c *Cache) setBacklinksOf(
	gen uint64, vaultID domain.VaultID, queried []string, fetched map[string][]domain.ResolvedLink,
) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generation != gen {
		return
	}
	v := c.getOrCreateVault(vaultID)
	for _, path := range queried {
		v.backlinksOf[path] = slices.Clone(fetched[path])
	}
}

func (c *Cache) copyNamedInto(vaultID domain.VaultID, names []string, dst map[string][]string) {
	if c == nil {
		return
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.vaults[vaultID]
	if !ok {
		return
	}
	for _, name := range names {
		if _, exists := dst[name]; exists {
			continue
		}
		if found, cached := v.named[name]; cached {
			dst[name] = slices.Clone(found)
		}
	}
}

func (c *Cache) setNamed(gen uint64, vaultID domain.VaultID, names []string, src map[string][]string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generation != gen {
		return
	}
	v := c.getOrCreateVault(vaultID)
	for _, name := range names {
		v.named[name] = slices.Clone(src[name])
	}
}

func (c *Cache) splitNotes(
	vaultID domain.VaultID, paths []string,
) (uint64, map[string]domain.NoteRef, []string) {
	if c == nil {
		return 0, map[string]domain.NoteRef{}, paths
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	gen := c.generation
	v, ok := c.vaults[vaultID]
	if !ok {
		return gen, map[string]domain.NoteRef{}, paths
	}
	cached := make(map[string]domain.NoteRef, len(paths))
	var missing []string
	seen := map[string]bool{}
	for _, path := range paths {
		if entry, found := v.notes[path]; found || v.isGraphLoaded {
			if entry.exists {
				cached[path] = entry.ref
			}
			continue
		}
		if !seen[path] {
			seen[path] = true
			missing = append(missing, path)
		}
	}
	return gen, cached, missing
}

func (c *Cache) setNotes(
	gen uint64, vaultID domain.VaultID, queried []string, fetched map[string]domain.NoteRef,
) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generation != gen {
		return
	}
	v := c.getOrCreateVault(vaultID)
	for _, path := range queried {
		ref, ok := fetched[path]
		v.notes[path] = cachedNoteRef{ref: ref, exists: ok}
	}
}

func (c *Cache) splitTypes(
	vaultID domain.VaultID, paths []string,
) (uint64, map[string]domain.NoteType, []string) {
	if c == nil {
		return 0, map[string]domain.NoteType{}, paths
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	gen := c.generation
	v, ok := c.vaults[vaultID]
	if !ok {
		return gen, map[string]domain.NoteType{}, paths
	}
	cached := make(map[string]domain.NoteType, len(paths))
	var missing []string
	seen := map[string]bool{}
	for _, path := range paths {
		if entry, found := v.types[path]; found || v.isGraphLoaded {
			if entry.exists {
				cached[path] = entry.noteType
			}
			continue
		}
		if !seen[path] {
			seen[path] = true
			missing = append(missing, path)
		}
	}
	return gen, cached, missing
}

func (c *Cache) setTypes(
	gen uint64, vaultID domain.VaultID, queried []string, fetched map[string]domain.NoteType,
) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generation != gen {
		return
	}
	v := c.getOrCreateVault(vaultID)
	for _, path := range queried {
		noteType, ok := fetched[path]
		v.types[path] = cachedNoteType{noteType: noteType, exists: ok}
	}
}

func (c *Cache) splitHeads(
	vaultID domain.VaultID, paths []string,
) (uint64, map[string]frontmatter.Head, []string, error) {
	if c == nil {
		return 0, map[string]frontmatter.Head{}, paths, nil
	}
	c.mu.RLock()
	gen := c.generation
	v, ok := c.vaults[vaultID]
	if !ok {
		c.mu.RUnlock()
		return gen, map[string]frontmatter.Head{}, paths, nil
	}
	cached := make(map[string]frontmatter.Head, len(paths))
	var missing []string
	var toDecode []string
	seen := map[string]bool{}
	isLoaded := v.isGraphLoaded
	for _, path := range paths {
		if entry, found := v.heads[path]; found {
			if entry.exists {
				head := entry.head
				if head.Values != nil {
					head.Values = maps.Clone(head.Values)
				}
				cached[path] = head
			}
			continue
		}
		if !seen[path] {
			seen[path] = true
			if isLoaded {
				if _, exists := v.types[path]; exists {
					toDecode = append(toDecode, path)
				}
			} else {
				missing = append(missing, path)
			}
		}
	}
	if len(toDecode) == 0 {
		c.mu.RUnlock()
		return gen, cached, missing, nil
	}
	raws := make(map[string]sql.NullString, len(toDecode))
	noteTypes := make(map[string]domain.NoteType, len(toDecode))
	for _, path := range toDecode {
		raws[path] = v.rawFrontmatter[path]
		noteTypes[path] = v.types[path].noteType
	}
	c.mu.RUnlock()

	decoded := make(map[string]frontmatter.Head, len(toDecode))
	for _, path := range toDecode {
		values, err := readHeadValues(raws[path])
		if err != nil {
			return 0, nil, nil, fmt.Errorf("%s: %w", path, err)
		}
		head := frontmatter.Head{Type: noteTypes[path], Values: values}
		decoded[path] = head
		cloned := head
		if cloned.Values != nil {
			cloned.Values = maps.Clone(cloned.Values)
		}
		cached[path] = cloned
	}
	c.setHeads(gen, vaultID, toDecode, decoded)
	return gen, cached, nil, nil
}

func (c *Cache) setHeads(
	gen uint64, vaultID domain.VaultID, queried []string, fetched map[string]frontmatter.Head,
) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generation != gen {
		return
	}
	v := c.getOrCreateVault(vaultID)
	for _, path := range queried {
		head, ok := fetched[path]
		if head.Values != nil {
			head.Values = maps.Clone(head.Values)
		}
		v.heads[path] = cachedHead{head: head, exists: ok}
	}
}
