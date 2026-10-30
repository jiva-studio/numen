package filetext

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
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// The two vaults every test here uses. Their books share no words, so a chunk
// that arrives from the wrong one is recognisable.
var (
	first  = domain.Vault{ID: "01FIRST", Name: "first", Path: "/first"}
	second = domain.Vault{ID: "01SECOND", Name: "second", Path: "/second"}
)

// The vocabularies the two vaults are written in.
var (
	sanskrit = []string{"udyana", "vrksa", "bija", "jala", "pathin", "grama", "nadi", "parvata"}
	latin    = []string{"aqua", "terra", "ventus", "ignis", "silva", "mons", "flumen", "campus"}
)

// store is the index, in memory, answering every question the way the index
// answers it. What the two passes do next is read from those answers.
type store struct {
	mu       sync.RWMutex
	sources  map[domain.VaultID]map[string]domain.Source // vault, then path
	chunks   []storedChunk
	vectors  map[domain.ChunkID][]embeddingport.Vector // the coarse rows, per chunk, appended, so a second write shows
	groups   [][]embeddingport.Vector                  // every write of vectors, in order
	outBytes map[string]int                            // extractions per path
	next     int64
	retained map[string][]byte
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
		sources:  map[domain.VaultID]map[string]domain.Source{},
		vectors:  map[domain.ChunkID][]embeddingport.Vector{},
		outBytes: map[string]int{},
	}
}

func (s *store) SaveSource(_ context.Context, vaultID domain.VaultID, src domain.Source) error {
	s.put(vaultID, src)
	return nil
}

func (s *store) SaveExtraction(_ context.Context, vaultID domain.VaultID, e domain.SourceChunks) error {
	s.outBytes[e.Source.FileStat.Path]++
	s.put(vaultID, e.Source)
	s.clear(vaultID, e.Source.FileStat.Path)

	for _, large := range e.Chunks {
		parent := s.insert(vaultID, e.Source.FileStat, large, 0)
		for _, small := range large.Small {
			s.insert(vaultID, e.Source.FileStat, small, parent)
		}
	}
	return nil
}

func (s *store) SaveVectors(_ context.Context, vectors []embeddingport.Vector) error {
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
	if s.retained == nil {
		return nil, nil
	}
	out := map[string][]byte{}
	for _, one := range of {
		if v, target := s.retained[recipe+"/"+hex.EncodeToString(one)]; target {
			out[hex.EncodeToString(one)] = v
		}
	}
	return out, nil
}

func (s *store) GetFileStats(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind) (map[string]domain.FileStat, error) {
	out := map[string]domain.FileStat{}
	for path, src := range s.sources[vaultID] {
		if src.FileStat.Kind != kind {
			continue
		}
		// The kind filter is already applied, so the result leaves the kind empty.
		out[path] = domain.FileStat{Path: path, Size: src.FileStat.Size, ModTime: src.FileStat.ModTime}
	}
	return out, nil
}

func (s *store) GetUnchunkedSources(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind, limit int) ([]string, error) {
	return s.paths(vaultID, kind, limit, func(src domain.Source) bool {
		return !s.cut(vaultID, src.FileStat.Path)
	})
}

func (s *store) GetSourcesWithOtherRecipe(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind, recipes []string, limit int) ([]string, error) {
	return s.paths(vaultID, kind, limit, func(src domain.Source) bool {
		return !slices.Contains(recipes, src.Recipe)
	})
}

func (s *store) GetUnembeddedChunks(_ context.Context, vaultID domain.VaultID, model embeddingport.EmbeddingModel, after embeddingport.ChunkCursor, limit int) ([]domain.Passage, embeddingport.ChunkCursor, error) {
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
		mtime := src.FileStat.ModTime.UnixNano()
		if mtime > afterMTime || (mtime == afterMTime && c.id <= afterChunk) {
			continue
		}
		// The source determines which text its chunks refer to, as the query does:
		// a chunk of a source with a recognition is read from that recognition and not
		// from the file.
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
		next = embeddingport.ChunkCursor(strconv.FormatInt(src.FileStat.ModTime.UnixNano(), 10) + ":" + strconv.FormatInt(lastChunk.id, 10))
	}
	return out, next, nil
}

// chunkID formats the row index as domain.ChunkID.
func chunkID(row int64) domain.ChunkID {
	return domain.ChunkID(strconv.FormatInt(row, 10))
}

// parseCursor is the point in the ordering a walk carries on after. The empty
// cursor starts from the beginning.
func parseCursor(after embeddingport.ChunkCursor) (int64, int64, error) {
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

// put records a source as it arrived, including the recipe: a source recorded
// without a recipe still requires its text.
func (s *store) put(vaultID domain.VaultID, src domain.Source) {
	if s.sources[vaultID] == nil {
		s.sources[vaultID] = map[string]domain.Source{}
	}
	s.sources[vaultID][src.FileStat.Path] = src
}

// clear takes out the chunks of one source, and the vectors made from them.
// RemoveSources takes the sources out, and their chunks and vectors with them,
// the way a foreign key does in the database.
func (s *store) RemoveSources(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind, paths []string) error {
	target := s.sources[vaultID]
	for _, path := range paths {
		if src, ok := target[path]; ok && src.FileStat.Kind == kind {
			delete(target, path)
			s.clear(vaultID, path)
		}
	}
	return nil
}

// MoveSources files what was at one path, and everything under it, where it now
// is. The chunks travel with the source, as they do in the database.
func (s *store) MoveSources(_ context.Context, vaultID domain.VaultID, from, to string) error {
	target := s.sources[vaultID]
	for path, src := range target {
		if path != from && !strings.HasPrefix(path, from+"/") {
			continue
		}
		landed := to + path[len(from):]
		delete(target, path)
		src.FileStat.Path = landed
		target[landed] = src
		for i, c := range s.chunks {
			if c.vault == vaultID && c.path == path {
				s.chunks[i].path = landed
			}
		}
	}
	return nil
}

// Under is every source the store holds at a path and beneath it.
func (s *store) GetSourcesUnder(_ context.Context, vaultID domain.VaultID, path string) ([]domain.FileStat, error) {
	var out []domain.FileStat
	for target, src := range s.sources[vaultID] {
		if target == path || strings.HasPrefix(target, path+"/") {
			out = append(out, src.FileStat)
		}
	}
	slices.SortFunc(out, func(a, b domain.FileStat) int { return strings.Compare(a.Path, b.Path) })
	return out, nil
}

func (s *store) clear(vaultID domain.VaultID, path string) {
	keptChunks := s.chunks[:0]
	for _, c := range s.chunks {
		if c.vault == vaultID && c.path == path {
			delete(s.vectors, chunkID(c.id))
			continue
		}
		keptChunks = append(keptChunks, c)
	}
	s.chunks = keptChunks
}

func (s *store) insert(vaultID domain.VaultID, ref domain.FileStat, c domain.Chunk, parent int64) int64 {
	s.next++
	s.chunks = append(s.chunks, storedChunk{
		id: s.next, vault: vaultID, path: ref.Path, kind: ref.Kind,
		start: c.Start, length: c.Length, location: c.Location, text: c.Text, parent: parent,
	})
	return s.next
}

// paths answers a question about sources as the paths that satisfy it, ordered
// and bounded the way the index orders and bounds them.
func (s *store) paths(vaultID domain.VaultID, kind domain.SourceKind, limit int, unembedded func(domain.Source) bool) ([]string, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("a question about sources needs a positive limit, got %d", limit)
	}
	var out []string
	for _, src := range s.sources[vaultID] {
		if src.FileStat.Kind == kind && unembedded(src) {
			out = append(out, src.FileStat.Path)
		}
	}
	slices.Sort(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// cut reports whether a source has a small chunk, which indicates it was cut.
func (s *store) cut(vaultID domain.VaultID, path string) bool {
	for _, c := range s.chunks {
		if c.vault == vaultID && c.path == path && c.parent != 0 {
			return true
		}
	}
	return false
}

// hasVector reports whether a chunk has a coarse row under the model, which the
// store writes once a vector is saved for it. A chunk without one needs a
// vector, regardless of other chunks with the same text.
func (s *store) hasVector(chunk int64, model embeddingport.EmbeddingModel) bool {
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
		timeA := srcA.FileStat.ModTime.UnixNano()
		timeB := srcB.FileStat.ModTime.UnixNano()
		if timeA != timeB {
			return cmp.Compare(timeB, timeA)
		}
		return cmp.Compare(a.id, b.id)
	})
	return out
}

// small is the chunks of one vault that carry a vector: the chunks a search
// runs over.
func (s *store) small(vaultID domain.VaultID) []storedChunk {
	var out []storedChunk
	for _, c := range s.getOrderedChunks() {
		if c.vault == vaultID && c.parent != 0 {
			out = append(out, c)
		}
	}
	return out
}

// documents reads the books bookOf writes.
type documents struct{}

// library is one vault as a set of files. It counts what was read, so a test can
// say that an unchanged file was not opened.
type library struct {
	// The interface is embedded for the listing, which no source scenario uses.
	vaultport.VaultReader
	files map[string]*shelved
	reads map[string]int
}

type shelved struct {
	kind domain.SourceKind
	raw  []byte
	// mtime is nanoseconds since the epoch, so a test can compare it against a
	// hand-written value.
	mtime int64
}

func newLibrary() *library {
	return &library{files: map[string]*shelved{}, reads: map[string]int{}}
}

// hold puts one file in the vault, at the modification time given.
func (l *library) hold(path string, kind domain.SourceKind, raw []byte, mtime int64) {
	l.files[path] = &shelved{kind: kind, raw: raw, mtime: mtime}
}

func (l *library) WalkFiles(_ context.Context, fn func(domain.FileStat) error) error {
	for _, path := range slices.Sorted(maps.Keys(l.files)) {
		if err := fn(l.ref(path)); err != nil {
			return err
		}
	}
	return nil
}

func (l *library) Read(_ context.Context, path string) ([]byte, error) {
	target, ok := l.files[path]
	if !ok {
		return nil, fmt.Errorf("read %s: %w", path, fs.ErrNotExist)
	}
	l.reads[path]++
	return target.raw, nil
}

func (l *library) Open(_ context.Context, path string) (io.ReadSeekCloser, error) {
	target, ok := l.files[path]
	if !ok {
		return nil, fmt.Errorf("open %s: %w", path, fs.ErrNotExist)
	}
	l.reads[path]++
	return nopCloser{bytes.NewReader(target.raw)}, nil
}

// nopCloser is a file held in memory, with nothing to release.
type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func (l *library) StatFile(_ context.Context, path string) (domain.FileStat, error) {
	if _, ok := l.files[path]; !ok {
		return domain.FileStat{}, fmt.Errorf("stat %s: %w", path, fs.ErrNotExist)
	}
	return l.ref(path), nil
}

func (l *library) ref(path string) domain.FileStat {
	target := l.files[path]
	return domain.FileStat{
		Path: path, Kind: target.kind, Size: int64(len(target.raw)), ModTime: time.Unix(0, target.mtime),
	}
}

// vaults opens the reader of each vault a test set up.
type vaults map[domain.VaultID]*library

func (v vaults) Open(vault domain.Vault) (vaultport.VaultReader, error) {
	reader, ok := v[vault.ID]
	if !ok {
		return nil, fmt.Errorf("no vault %s", vault.ID)
	}
	return reader, nil
}

// words returns n words of one vocabulary. A vault's books use their own
// vocabulary, so a chunk identifies the vault it came from.
func words(vocabulary []string, n int) string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, vocabulary[i%len(vocabulary)])
	}
	return strings.Join(out, " ")
}

func (s *store) GetSourceText(_ context.Context, vaultID domain.VaultID, path string) (sourceport.SourceText, bool, error) {
	src, target := s.sources[vaultID][path]
	if !target {
		return sourceport.SourceText{}, false, nil
	}
	// The row records the file state at read time, as the query does.
	return sourceport.SourceText{
		FileStat: domain.FileStat{
			Path: path, Size: src.FileStat.Size, ModTime: src.FileStat.ModTime,
		},
		Producer: src.Producer,
		Hash:     src.Hash,
	}, true, nil
}

func (s *store) GetRecognisedSources(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind) ([]sourceport.SourceText, error) {
	var out []sourceport.SourceText
	for path, src := range s.sources[vaultID] {
		if src.FileStat.Kind == kind && src.Producer != "" {
			out = append(out, sourceport.SourceText{
				FileStat: domain.FileStat{Path: path},
				Producer: src.Producer,
				Hash:     src.Hash,
			})
		}
	}
	slices.SortFunc(out, func(a, b sourceport.SourceText) int { return cmp.Compare(a.Path, b.Path) })
	return out, nil
}

// bookOf returns the bytes of a book whose spine is the given parts, each under
// its own heading, so the book has distinct locations.
func bookOf(t testing.TB, title string, parts ...string) []byte {
	t.Helper()
	return []byte(bookMark + title + bookSeparator + strings.Join(parts, bookSeparator))
}

// The bytes a book is written as in these tests: the mark, the title, and the
// parts, each after a separator.
const (
	bookMark      = "%BOOK\n"
	bookSeparator = "\x1f"
)

// ReadBook reads the bytes bookOf gives: each part is a heading and its words,
// and the book names each heading.
func (documents) ReadBook(_ context.Context, raw []byte) (bookText, error) {
	if !bytes.HasPrefix(raw, []byte(bookMark)) {
		return bookText{}, sourceport.ErrNotADocument
	}
	fields := strings.Split(strings.TrimPrefix(string(raw), bookMark), bookSeparator)
	var out bookText
	var text strings.Builder
	for i, part := range fields[1:] {
		title := fmt.Sprintf("Part %d", i+1)
		out.Parts = append(out.Parts, domain.PartStart{Title: title, Offset: text.Len()})
		text.WriteString(title + "\n" + part + "\n")
	}
	out.Text = text.String()
	return out, nil
}

// newIndex is the cut of a library whose books have the text versions the
// stand-in reads.
func newIndex(shelves vaults, sources sourceport.SourceRepository, queries sourceport.SourceQueries) Index {
	extract, _ := newIndexWithTexts(shelves, sources, queries)
	return extract
}

// newIndexWithTexts is newIndex, and the text versions a test puts more of.
func newIndexWithTexts(shelves vaults, sources sourceport.SourceRepository, queries sourceport.SourceQueries) (Index, *libraryTexts) {
	texts := newLibraryTexts(shelves)
	return NewIndex(shelves, sources, queries, texts), texts
}

// bookText is a book read: its text, and the parts it names.
type bookText struct {
	Text  string
	Parts []domain.PartStart
}
