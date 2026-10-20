package transcription

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// The two vaults every test here uses. Their books share no words, so a chunk
// that arrives from the wrong one is recognisable.
var (
	first = domain.Vault{ID: "01FIRST", Name: "first", Path: "/first"}
)

// store is the index, in memory, answering every question the way the index
// answers it. What the two passes do next is read from those answers.
type store struct {
	mu      sync.RWMutex
	sources map[domain.VaultID]map[string]domain.Source // vault, then path
	chunks  []storedChunk
	vectors map[domain.ChunkID][]port.Vector // the coarse rows, by chunk, appended, so a second write shows
	groups  [][]port.Vector                  // every write of vectors, in order
	written map[string]int                   // extractions per path
	next    int64
	// kept is what the index already holds, by recipe and hash. A test seeds it
	// to stand for vectors bought before the run under test.
	kept map[string][]byte
}

// storedChunk is one row of chunks. `parent` is zero for a large chunk.
type storedChunk struct {
	id       int64
	vault    domain.VaultID
	path     string
	kind     domain.SourceKind
	start    int
	length   int
	location string
	text     string
	parent   int64
}

func newStore() *store {
	return &store{
		sources: map[domain.VaultID]map[string]domain.Source{},
		vectors: map[domain.ChunkID][]port.Vector{},
		written: map[string]int{},
	}
}

func (s *store) SaveSource(_ context.Context, vaultID domain.VaultID, src domain.Source) error {
	s.put(vaultID, src)
	return nil
}

func (s *store) SaveExtraction(_ context.Context, vaultID domain.VaultID, e domain.SourceChunks) error {
	s.written[e.Source.Fingerprint.Path]++
	s.put(vaultID, e.Source)
	s.clear(vaultID, e.Source.Fingerprint.Path)

	for _, large := range e.Chunks {
		parent := s.insert(vaultID, e.Source.Fingerprint, large, 0)
		for _, small := range large.Small {
			s.insert(vaultID, e.Source.Fingerprint, small, parent)
		}
	}
	return nil
}

func (s *store) SaveVectors(_ context.Context, vectors []port.Vector) error {
	if len(vectors) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groups = append(s.groups, slices.Clone(vectors))
	for _, v := range vectors {
		s.vectors[v.ChunkID] = append(s.vectors[v.ChunkID], v)
	}
	return nil
}

// GetKeptVectors is the vectors this store already holds for the texts given,
// which is what a real index answers out of what it was paid for.
func (s *store) GetKeptVectors(_ context.Context, recipe string, of [][]byte) (map[string][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.kept == nil {
		return nil, nil
	}
	out := map[string][]byte{}
	for _, one := range of {
		if v, held := s.kept[recipe+"/"+hex.EncodeToString(one)]; held {
			out[hex.EncodeToString(one)] = v
		}
	}
	return out, nil
}

func (s *store) GetFingerprints(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind) (map[string]domain.Fingerprint, error) {
	out := map[string]domain.Fingerprint{}
	for path, src := range s.sources[vaultID] {
		if src.Fingerprint.Kind != kind {
			continue
		}
		// The kind is what was asked for, so the answer leaves it empty.
		out[path] = domain.Fingerprint{Path: path, Size: src.Fingerprint.Size, ModTime: src.Fingerprint.ModTime}
	}
	return out, nil
}

func (s *store) GetUnchunkedSources(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind, limit int) ([]string, error) {
	return s.paths(vaultID, kind, limit, func(src domain.Source) bool {
		return !s.cut(vaultID, src.Fingerprint.Path)
	})
}

func (s *store) GetSourcesWithOtherRecipe(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind, recipes []string, limit int) ([]string, error) {
	return s.paths(vaultID, kind, limit, func(src domain.Source) bool {
		return !slices.Contains(recipes, src.Recipe)
	})
}

func (s *store) GetUnembeddedChunks(_ context.Context, vaultID domain.VaultID, model port.EmbeddingModel, after port.ChunkCursor, limit int) ([]domain.Passage, port.ChunkCursor, error) {
	if limit <= 0 {
		return nil, "", fmt.Errorf("a batch needs a positive limit, got %d", limit)
	}
	afterMTime, afterChunk, err := parseCursor(after)
	if err != nil {
		return nil, "", err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.Passage
	var lastChunk storedChunk
	for _, c := range s.getOrderedChunks() {
		if c.vault != vaultID || c.parent == 0 || s.hasVector(c.id, model) {
			continue
		}
		src := s.sources[c.vault][c.path]
		mtime := src.Fingerprint.ModTime.UnixNano()
		if mtime > afterMTime || (mtime == afterMTime && c.id <= afterChunk) {
			continue
		}
		// The source says which text its chunks are places in, as the query
		// does: a chunk of a source standing on a recognition is read from that
		// reading and not from the file.
		out = append(out, domain.Passage{
			ChunkID: chunkID(c.id), Source: c.path, Start: c.start, Length: c.length, Location: c.location,
			Producer: src.Producer, SourceHash: src.Hash, ChunkHash: hashOf(c.text),
		})
		lastChunk = c
		if len(out) == limit {
			break
		}
	}
	next := after
	if len(out) > 0 {
		src := s.sources[lastChunk.vault][lastChunk.path]
		next = port.ChunkCursor(strconv.FormatInt(src.Fingerprint.ModTime.UnixNano(), 10) + ":" + strconv.FormatInt(lastChunk.id, 10))
	}
	return out, next, nil
}

// chunkID is how this store spells the row a chunk sits on, as an index does.
func chunkID(row int64) domain.ChunkID {
	return domain.ChunkID(strconv.FormatInt(row, 10))
}

// parseCursor is the point in the ordering a walk carries on after. The empty
// cursor starts from the beginning.
func parseCursor(after port.ChunkCursor) (int64, int64, error) {
	if after == "" {
		return math.MaxInt64, 0, nil
	}
	mtimeStr, chunkStr, ok := strings.Cut(string(after), ":")
	if !ok {
		row, err := strconv.ParseInt(string(after), 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("%q is no chunk of this store", after)
		}
		return math.MaxInt64, row, nil
	}
	mtime, err := strconv.ParseInt(mtimeStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%q is no cursor of this store", after)
	}
	chunkRow, err := strconv.ParseInt(chunkStr, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%q is no cursor of this store", after)
	}
	return mtime, chunkRow, nil
}

// hashOf is the address an index gives a chunk's text, and what a vector
// already bought is reclaimed by.
func hashOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// put records a source as it arrived, the recipe included: a source recorded
// without one owes its text.
func (s *store) put(vaultID domain.VaultID, src domain.Source) {
	if s.sources[vaultID] == nil {
		s.sources[vaultID] = map[string]domain.Source{}
	}
	s.sources[vaultID][src.Fingerprint.Path] = src
}

// clear takes out the chunks of one source, and the vectors made from them.
// RemoveSources takes the sources out, and their chunks and vectors with them,
// the way a foreign key does in the database.
func (s *store) RemoveSources(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind, paths []string) error {
	held := s.sources[vaultID]
	for _, path := range paths {
		if src, ok := held[path]; ok && src.Fingerprint.Kind == kind {
			delete(held, path)
			s.clear(vaultID, path)
		}
	}
	return nil
}

// MoveSources files what was at one path, and everything under it, where it now
// is. The chunks travel with the source, as they do in the database.
func (s *store) MoveSources(_ context.Context, vaultID domain.VaultID, from, to string) error {
	held := s.sources[vaultID]
	for path, src := range held {
		if path != from && !strings.HasPrefix(path, from+"/") {
			continue
		}
		landed := to + path[len(from):]
		delete(held, path)
		src.Fingerprint.Path = landed
		held[landed] = src
		for i, c := range s.chunks {
			if c.vault == vaultID && c.path == path {
				s.chunks[i].path = landed
			}
		}
	}
	return nil
}

// Under is every source the store holds at a path and beneath it.
func (s *store) GetSourcesUnder(_ context.Context, vaultID domain.VaultID, path string) ([]domain.Fingerprint, error) {
	var out []domain.Fingerprint
	for held, src := range s.sources[vaultID] {
		if held == path || strings.HasPrefix(held, path+"/") {
			out = append(out, src.Fingerprint)
		}
	}
	slices.SortFunc(out, func(a, b domain.Fingerprint) int { return strings.Compare(a.Path, b.Path) })
	return out, nil
}

func (s *store) clear(vaultID domain.VaultID, path string) {
	kept := s.chunks[:0]
	for _, c := range s.chunks {
		if c.vault == vaultID && c.path == path {
			delete(s.vectors, chunkID(c.id))
			continue
		}
		kept = append(kept, c)
	}
	s.chunks = kept
}

func (s *store) insert(vaultID domain.VaultID, ref domain.Fingerprint, c domain.Chunk, parent int64) int64 {
	s.next++
	s.chunks = append(s.chunks, storedChunk{
		id: s.next, vault: vaultID, path: ref.Path, kind: ref.Kind,
		start: c.Start, length: c.Length, location: c.Location, text: c.Text, parent: parent,
	})
	return s.next
}

// paths answers a question about sources as the paths that satisfy it, ordered
// and bounded the way the index orders and bounds them.
func (s *store) paths(vaultID domain.VaultID, kind domain.SourceKind, limit int, owing func(domain.Source) bool) ([]string, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("a question about sources needs a positive limit, got %d", limit)
	}
	var out []string
	for _, src := range s.sources[vaultID] {
		if src.Fingerprint.Kind == kind && owing(src) {
			out = append(out, src.Fingerprint.Path)
		}
	}
	slices.Sort(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// cut says whether a source has a small chunk, which is what having been cut
// means.
func (s *store) cut(vaultID domain.VaultID, path string) bool {
	for _, c := range s.chunks {
		if c.vault == vaultID && c.path == path && c.parent != 0 {
			return true
		}
	}
	return false
}

// hasVector says whether a chunk has a coarse row under the model, which is
// what the store writes once a vector is saved for it. A chunk without one owes,
// whatever other chunks hold the same text.
func (s *store) hasVector(chunk int64, model port.EmbeddingModel) bool {
	for _, v := range s.vectors[chunkID(chunk)] {
		if v.Model == model {
			return true
		}
	}
	return false
}

// getOrderedChunks is the chunks ordered by source modification time descending,
// then chunk id ascending.
func (s *store) getOrderedChunks() []storedChunk {
	out := slices.Clone(s.chunks)
	slices.SortFunc(out, func(a, b storedChunk) int {
		srcA := s.sources[a.vault][a.path]
		srcB := s.sources[b.vault][b.path]
		timeA := srcA.Fingerprint.ModTime.UnixNano()
		timeB := srcB.Fingerprint.ModTime.UnixNano()
		if timeA != timeB {
			return cmp.Compare(timeB, timeA)
		}
		return cmp.Compare(a.id, b.id)
	})
	return out
}

// library is one vault as a set of files. It counts what was read, so a test can
// say that an unchanged file was not opened.
type library struct {
	// The interface is embedded for the listing, which no source scenario asks
	// for.
	port.VaultReader
	files map[string]*shelved
	reads map[string]int
}

type shelved struct {
	kind domain.SourceKind
	raw  []byte
	// mtime is nanoseconds since the epoch, which is what a test writing one
	// out by hand can compare against.
	mtime int64
}

func newLibrary() *library {
	return &library{files: map[string]*shelved{}, reads: map[string]int{}}
}

// hold puts one file in the vault, at the modification time given.
func (l *library) hold(path string, kind domain.SourceKind, raw []byte, mtime int64) {
	l.files[path] = &shelved{kind: kind, raw: raw, mtime: mtime}
}

func (l *library) WalkFiles(_ context.Context, fn func(domain.Fingerprint) error) error {
	for _, path := range slices.Sorted(maps.Keys(l.files)) {
		if err := fn(l.ref(path)); err != nil {
			return err
		}
	}
	return nil
}

func (l *library) Read(_ context.Context, path string) ([]byte, error) {
	held, ok := l.files[path]
	if !ok {
		return nil, fmt.Errorf("read %s: %w", path, fs.ErrNotExist)
	}
	l.reads[path]++
	return held.raw, nil
}

func (l *library) Open(_ context.Context, path string) (io.ReadSeekCloser, error) {
	held, ok := l.files[path]
	if !ok {
		return nil, fmt.Errorf("open %s: %w", path, fs.ErrNotExist)
	}
	l.reads[path]++
	return nopCloser{bytes.NewReader(held.raw)}, nil
}

// nopCloser is a file held in memory, with nothing to release.
type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func (l *library) GetFingerprint(_ context.Context, path string) (domain.Fingerprint, error) {
	if _, ok := l.files[path]; !ok {
		return domain.Fingerprint{}, fmt.Errorf("stat %s: %w", path, fs.ErrNotExist)
	}
	return l.ref(path), nil
}

func (l *library) ref(path string) domain.Fingerprint {
	held := l.files[path]
	return domain.Fingerprint{
		Path: path, Kind: held.kind, Size: int64(len(held.raw)), ModTime: time.Unix(0, held.mtime),
	}
}

// vaults opens the reader of each vault a test set up.
type vaults map[domain.VaultID]*library

func (v vaults) Open(vault domain.Vault) (port.VaultReader, error) {
	reader, ok := v[vault.ID]
	if !ok {
		return nil, fmt.Errorf("no vault %s", vault.ID)
	}
	return reader, nil
}

func (s *store) GetSourceText(_ context.Context, vaultID domain.VaultID, path string) (port.SourceText, bool, error) {
	src, held := s.sources[vaultID][path]
	if !held {
		return port.SourceText{}, false, nil
	}
	// The row says what the file was when it was read, as the query does.
	return port.SourceText{
		Fingerprint: domain.Fingerprint{
			Path: path, Size: src.Fingerprint.Size, ModTime: src.Fingerprint.ModTime,
		},
		Producer: src.Producer,
		Hash:     src.Hash,
	}, true, nil
}

func (s *store) GetRecognisedSources(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind) ([]port.SourceText, error) {
	var out []port.SourceText
	for path, src := range s.sources[vaultID] {
		if src.Fingerprint.Kind == kind && src.Producer != "" {
			out = append(out, port.SourceText{
				Fingerprint: domain.Fingerprint{Path: path},
				Producer:    src.Producer,
				Hash:        src.Hash,
			})
		}
	}
	slices.SortFunc(out, func(a, b port.SourceText) int { return cmp.Compare(a.Path, b.Path) })
	return out, nil
}

func (l *library) OpenShared(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	return l.Open(ctx, path)
}
