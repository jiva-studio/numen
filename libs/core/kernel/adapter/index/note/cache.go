package note

import (
	"context"
	"database/sql"
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

type headCacheRead struct {
	cached    map[string]frontmatter.Head
	missing   []string
	toDecode  []string
	raws      map[string]sql.NullString
	noteTypes map[string]domain.NoteType
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
	vault, found, err := getGraphVault(ctx, c, db, gen, vaultID)
	if err != nil || !found {
		return err
	}
	graph, err := loadVaultGraph(ctx, db, vault, vaultID)
	if err != nil {
		return err
	}
	setVaultGraph(c, gen, vaultID, vault, graph)
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
	read := readCachedHeads(v, paths)
	if len(read.toDecode) == 0 {
		c.mu.RUnlock()
		return gen, read.cached, read.missing, nil
	}
	c.mu.RUnlock()
	decoded, err := decodeHeads(read)
	if err != nil {
		return 0, nil, nil, err
	}
	for path, head := range decoded {
		read.cached[path] = cloneHead(head)
	}
	c.setHeads(gen, vaultID, read.toDecode, decoded)
	return gen, read.cached, nil, nil
}

func readCachedHeads(v *vaultCache, paths []string) headCacheRead {
	read := headCacheRead{cached: make(map[string]frontmatter.Head, len(paths)), raws: map[string]sql.NullString{}, noteTypes: map[string]domain.NoteType{}}
	seen := map[string]bool{}
	for _, path := range paths {
		if head, found, exists := getCachedHead(v, path); found {
			if exists {
				read.cached[path] = head
			}
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		read.addHeadPath(v, path)
	}
	return read
}

func getCachedHead(v *vaultCache, path string) (frontmatter.Head, bool, bool) {
	entry, found := v.heads[path]
	if !found {
		return frontmatter.Head{}, false, false
	}
	if !entry.exists {
		return frontmatter.Head{}, true, false
	}
	return cloneHead(entry.head), true, true
}

func (r *headCacheRead) addHeadPath(v *vaultCache, path string) {
	if !v.isGraphLoaded {
		r.missing = append(r.missing, path)
		return
	}
	noteType, exists := v.types[path]
	if !exists {
		return
	}
	r.toDecode = append(r.toDecode, path)
	r.raws[path] = v.rawFrontmatter[path]
	r.noteTypes[path] = noteType.noteType
}

func decodeHeads(read headCacheRead) (map[string]frontmatter.Head, error) {
	decoded := make(map[string]frontmatter.Head, len(read.toDecode))
	for _, path := range read.toDecode {
		values, err := readHeadValues(read.raws[path])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		decoded[path] = frontmatter.Head{Type: read.noteTypes[path], Values: values}
	}
	return decoded, nil
}

func cloneHead(head frontmatter.Head) frontmatter.Head {
	if head.Values != nil {
		head.Values = maps.Clone(head.Values)
	}
	return head
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
