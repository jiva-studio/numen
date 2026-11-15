// Package indexfake is the index, a vault and its books in memory, for the
// tests of the use cases that cut and embed sources.
package indexfake

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
)

// Store is the index, in memory, answering every question the way the index
// answers it.
type Store struct {
	mu          sync.RWMutex
	Sources     map[domain.VaultID]map[string]domain.Source
	Chunks      []Chunk
	Vectors     map[domain.ChunkID][]vector.Stored
	Groups      [][]vector.Stored
	Extractions map[string]int
	next        int64
	Kept        map[string][]byte
}

// Chunk is one row of chunks. Parent is zero for a large chunk.
type Chunk struct {
	ID       int64
	Vault    domain.VaultID
	Path     string
	Kind     domain.SourceKind
	Start    int
	Length   int
	Location string
	Text     string
	Parent   int64
}

func NewStore() *Store {
	return &Store{
		Sources:     map[domain.VaultID]map[string]domain.Source{},
		Vectors:     map[domain.ChunkID][]vector.Stored{},
		Extractions: map[string]int{},
	}
}

func (s *Store) SaveSource(_ context.Context, vaultID domain.VaultID, src domain.Source) error {
	s.put(vaultID, src)
	return nil
}

func (s *Store) SaveExtraction(_ context.Context, vaultID domain.VaultID, e domain.SourceChunks) error {
	s.Extractions[e.Source.FileStat.Path]++
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

func (s *Store) SaveVectors(_ context.Context, vectors []vector.Stored) error {
	if len(vectors) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Groups = append(s.Groups, slices.Clone(vectors))
	for _, v := range vectors {
		s.Vectors[v.ChunkID] = append(s.Vectors[v.ChunkID], v)
	}
	return nil
}

// GetKeptVectors returns the vectors in Kept for the texts given.
func (s *Store) GetKeptVectors(_ context.Context, recipe string, of [][]byte) (map[string][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Kept == nil {
		return nil, nil
	}
	out := map[string][]byte{}
	for _, one := range of {
		if v, ok := s.Kept[recipe+"/"+hex.EncodeToString(one)]; ok {
			out[hex.EncodeToString(one)] = v
		}
	}
	return out, nil
}

func (s *Store) GetFileStats(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind) (map[string]domain.FileStat, error) {
	out := map[string]domain.FileStat{}
	for path, src := range s.Sources[vaultID] {
		if src.FileStat.Kind != kind {
			continue
		}
		// The kind filter is already applied, so the result leaves the kind empty.
		out[path] = domain.FileStat{Path: path, Size: src.FileStat.Size, ModTime: src.FileStat.ModTime}
	}
	return out, nil
}

func (s *Store) GetUnchunkedSources(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind, limit int) ([]string, error) {
	return s.getPaths(vaultID, kind, limit, func(src domain.Source) bool {
		return !s.IsCut(vaultID, src.FileStat.Path)
	})
}

func (s *Store) GetSourcesWithOtherRecipe(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind, recipes []string, limit int) ([]string, error) {
	return s.getPaths(vaultID, kind, limit, func(src domain.Source) bool {
		return !slices.Contains(recipes, src.Recipe)
	})
}

func (s *Store) GetUnembeddedChunks(_ context.Context, vaultID domain.VaultID, recipe string, after vector.ChunkCursor, limit int) ([]domain.SearchResult, vector.ChunkCursor, error) {
	if limit <= 0 {
		return nil, "", fmt.Errorf("a batch needs a positive limit, got %d", limit)
	}
	afterMTime, afterChunk, err := parseCursor(after)
	if err != nil {
		return nil, "", err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []domain.SearchResult
	var lastChunk Chunk
	for _, c := range s.getOrderedChunks() {
		if c.Vault != vaultID || c.Parent == 0 || s.hasVector(c.ID, recipe) {
			continue
		}
		src := s.Sources[c.Vault][c.Path]
		mtime := src.FileStat.ModTime.UnixNano()
		if mtime > afterMTime || (mtime == afterMTime && c.ID <= afterChunk) {
			continue
		}
		// The source determines which text its chunks refer to, as the query does:
		// a chunk of a source with a recognition is read from that recognition and
		// not from the file.
		out = append(out, domain.SearchResult{
			ChunkID: GetChunkID(c.ID), Path: c.Path, Start: c.Start, Length: c.Length, PlaceLabel: c.Location,
			VersionID: src.Producer, FileHash: src.ContentHash, ChunkHash: GetHash(c.Text),
		})
		lastChunk = c
		if len(out) == limit {
			break
		}
	}
	next := after
	if len(out) > 0 {
		src := s.Sources[lastChunk.Vault][lastChunk.Path]
		next = vector.ChunkCursor(strconv.FormatInt(src.FileStat.ModTime.UnixNano(), 10) + ":" + strconv.FormatInt(lastChunk.ID, 10))
	}
	return out, next, nil
}

func GetChunkID(row int64) domain.ChunkID {
	return domain.ChunkID(strconv.FormatInt(row, 10))
}

func parseCursor(after vector.ChunkCursor) (int64, int64, error) {
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

// GetHash returns the hash an index gives a chunk's text, by which a vector
// computed earlier is reused.
func GetHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func (s *Store) put(vaultID domain.VaultID, src domain.Source) {
	if s.Sources[vaultID] == nil {
		s.Sources[vaultID] = map[string]domain.Source{}
	}
	s.Sources[vaultID][src.FileStat.Path] = src
}

// RemoveSources removes the sources, and their chunks and vectors with them,
// as the foreign keys of the database do.
func (s *Store) RemoveSources(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind, paths []string) error {
	target := s.Sources[vaultID]
	for _, path := range paths {
		if src, ok := target[path]; ok && src.FileStat.Kind == kind {
			delete(target, path)
			s.clear(vaultID, path)
		}
	}
	return nil
}

// MoveSources moves the source at one path, and every source under it, to the
// new path. The chunks move with the source, as they do in the database.
func (s *Store) MoveSources(_ context.Context, vaultID domain.VaultID, from, to string) error {
	target := s.Sources[vaultID]
	for path, src := range target {
		if path != from && !strings.HasPrefix(path, from+"/") {
			continue
		}
		moved := to + path[len(from):]
		delete(target, path)
		src.FileStat.Path = moved
		target[moved] = src
		for i, c := range s.Chunks {
			if c.Vault == vaultID && c.Path == path {
				s.Chunks[i].Path = moved
			}
		}
	}
	return nil
}

// GetSourcesUnder returns every source at a path and beneath it.
func (s *Store) GetSourcesUnder(_ context.Context, vaultID domain.VaultID, path string) ([]domain.FileStat, error) {
	var out []domain.FileStat
	for target, src := range s.Sources[vaultID] {
		if target == path || strings.HasPrefix(target, path+"/") {
			out = append(out, src.FileStat)
		}
	}
	slices.SortFunc(out, func(a, b domain.FileStat) int { return strings.Compare(a.Path, b.Path) })
	return out, nil
}

func (s *Store) GetSourceText(_ context.Context, vaultID domain.VaultID, path string) (source.Text, bool, error) {
	src, ok := s.Sources[vaultID][path]
	if !ok {
		return source.Text{}, false, nil
	}
	// The row records the file state at read time, as the query does.
	return source.Text{
		FileStat: domain.FileStat{
			Path: path, Size: src.FileStat.Size, ModTime: src.FileStat.ModTime,
		},
		Producer:    src.Producer,
		ContentHash: src.ContentHash,
	}, true, nil
}

func (s *Store) GetRecognizedSources(_ context.Context, vaultID domain.VaultID, kind domain.SourceKind) ([]source.Text, error) {
	var out []source.Text
	for path, src := range s.Sources[vaultID] {
		if src.FileStat.Kind == kind && src.Producer != "" {
			out = append(out, source.Text{
				FileStat:    domain.FileStat{Path: path},
				Producer:    src.Producer,
				ContentHash: src.ContentHash,
			})
		}
	}
	slices.SortFunc(out, func(a, b source.Text) int { return cmp.Compare(a.Path, b.Path) })
	return out, nil
}

func (s *Store) clear(vaultID domain.VaultID, path string) {
	keptChunks := s.Chunks[:0]
	for _, c := range s.Chunks {
		if c.Vault == vaultID && c.Path == path {
			delete(s.Vectors, GetChunkID(c.ID))
			continue
		}
		keptChunks = append(keptChunks, c)
	}
	s.Chunks = keptChunks
}

func (s *Store) insert(vaultID domain.VaultID, ref domain.FileStat, c domain.Chunk, parent int64) int64 {
	s.next++
	s.Chunks = append(s.Chunks, Chunk{
		ID: s.next, Vault: vaultID, Path: ref.Path, Kind: ref.Kind,
		Start: c.Start, Length: c.Length, Location: c.Location, Text: c.Text, Parent: parent,
	})
	return s.next
}

func (s *Store) getPaths(vaultID domain.VaultID, kind domain.SourceKind, limit int, isMatch func(domain.Source) bool) ([]string, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("a question about sources needs a positive limit, got %d", limit)
	}
	var out []string
	for _, src := range s.Sources[vaultID] {
		if src.FileStat.Kind == kind && isMatch(src) {
			out = append(out, src.FileStat.Path)
		}
	}
	slices.Sort(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// IsCut reports whether a source has a small chunk, which means it was cut.
func (s *Store) IsCut(vaultID domain.VaultID, path string) bool {
	for _, c := range s.Chunks {
		if c.Vault == vaultID && c.Path == path && c.Parent != 0 {
			return true
		}
	}
	return false
}

// HasChunk reports whether the index contains the given chunk.
func (s *Store) HasChunk(chunk domain.ChunkID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.Chunks {
		if GetChunkID(c.ID) == chunk {
			return true
		}
	}
	return false
}

func (s *Store) GetVectors(c Chunk) []vector.Stored {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Clone(s.Vectors[GetChunkID(c.ID)])
}

func (s *Store) hasVector(chunk int64, recipe string) bool {
	for _, v := range s.Vectors[GetChunkID(chunk)] {
		if v.Recipe == recipe {
			return true
		}
	}
	return false
}

func (s *Store) getOrderedChunks() []Chunk {
	out := slices.Clone(s.Chunks)
	slices.SortFunc(out, func(a, b Chunk) int {
		srcA := s.Sources[a.Vault][a.Path]
		srcB := s.Sources[b.Vault][b.Path]
		timeA := srcA.FileStat.ModTime.UnixNano()
		timeB := srcB.FileStat.ModTime.UnixNano()
		if timeA != timeB {
			return cmp.Compare(timeB, timeA)
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return out
}

// GetSmallChunks returns the chunks of one vault that have a vector: the
// chunks a search runs over.
func (s *Store) GetSmallChunks(vaultID domain.VaultID) []Chunk {
	var out []Chunk
	for _, c := range s.getOrderedChunks() {
		if c.Vault == vaultID && c.Parent != 0 {
			out = append(out, c)
		}
	}
	return out
}

var (
	_ sourceport.SourcesUnder  = (*Store)(nil)
	_ sourceport.RemovalLookup = (*Store)(nil)
	_ sourceport.SourceRecords = (*Store)(nil)
	_ sourceport.SourceBacklog = (*Store)(nil)
)
