package source

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io/fs"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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
	sources map[string]map[string]port.Source // vault, then path
	chunks  []storedChunk
	vectors map[int64][]port.Vector // by chunk, appended, so a second write shows
	groups  [][]port.Vector         // every write of vectors, in order
	written map[string]int          // extractions per path
	next    int64
	// kept is what the index already holds, by recipe and fingerprint.
	kept map[string][]byte
}

// storedChunk is one row of chunks. `parent` is zero for a large chunk.
type storedChunk struct {
	id       int64
	vault    string
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
		sources: map[string]map[string]port.Source{},
		vectors: map[int64][]port.Vector{},
		written: map[string]int{},
	}
}

func (s *store) SaveSource(_ context.Context, vaultID string, src port.Source) error {
	s.put(vaultID, src)
	return nil
}

func (s *store) SaveExtraction(_ context.Context, vaultID string, e port.SourceChunks) error {
	s.written[e.Source.Ref.Path]++
	s.put(vaultID, e.Source)
	s.clear(vaultID, e.Source.Ref.Path)

	for _, large := range e.Chunks {
		parent := s.insert(vaultID, e.Source.Ref, large, 0)
		for _, small := range large.Small {
			s.insert(vaultID, e.Source.Ref, small, parent)
		}
	}
	return nil
}

func (s *store) SaveVectors(_ context.Context, vectors []port.Vector) error {
	if len(vectors) == 0 {
		return nil
	}
	s.groups = append(s.groups, slices.Clone(vectors))
	for _, v := range vectors {
		s.vectors[v.ChunkID] = append(s.vectors[v.ChunkID], v)
	}
	return nil
}

// Kept is the vectors this store already holds for the texts given, which is
// what a real index answers out of what it was paid for.
func (s *store) Kept(_ context.Context, recipe string, of [][]byte) (map[string][]byte, error) {
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

func (s *store) Fingerprints(_ context.Context, vaultID string, kind domain.SourceKind) (map[string]domain.Fingerprint, error) {
	out := map[string]domain.Fingerprint{}
	for path, src := range s.sources[vaultID] {
		if src.Ref.Kind != kind {
			continue
		}
		// The kind is what was asked for, so the answer leaves it empty.
		out[path] = domain.Fingerprint{Path: path, Size: src.Ref.Size, MTime: src.Ref.MTime}
	}
	return out, nil
}

func (s *store) Unchunked(_ context.Context, vaultID string, kind domain.SourceKind, limit int) ([]string, error) {
	return s.paths(vaultID, kind, limit, func(src port.Source) bool {
		return !s.cut(vaultID, src.Ref.Path)
	})
}

func (s *store) ByOtherRecipe(_ context.Context, vaultID string, kind domain.SourceKind, recipes []string, limit int) ([]string, error) {
	return s.paths(vaultID, kind, limit, func(src port.Source) bool {
		return !slices.Contains(recipes, src.Recipe)
	})
}

func (s *store) Unembedded(_ context.Context, vaultID string, model port.EmbeddingModel, after int64, limit int) ([]domain.Passage, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("a batch needs a positive limit, got %d", limit)
	}
	var out []domain.Passage
	for _, c := range s.ordered() {
		if c.vault != vaultID || c.parent == 0 || c.id <= after || s.embedded(c.id, model) {
			continue
		}
		// The source says which text its chunks are places in, as the query
		// does: a chunk of a source standing on a reading is read from that
		// reading and not from the file.
		src := s.sources[c.vault][c.path]
		out = append(out, domain.Passage{
			ChunkID: c.id, Source: c.path, Start: c.start, Length: c.length, Location: c.location,
			TextFrom: src.TextFrom, Hash: src.Hash, Fingerprint: fingerprintOf(c.text),
		})
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

// fingerprintOf is what an index records a chunk's text as, and what a vector
// already bought is reclaimed by.
func fingerprintOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// put records a source as it arrived, the recipe included: a source recorded
// without one owes its text.
func (s *store) put(vaultID string, src port.Source) {
	if s.sources[vaultID] == nil {
		s.sources[vaultID] = map[string]port.Source{}
	}
	s.sources[vaultID][src.Ref.Path] = src
}

// clear takes out the chunks of one source, and the vectors made from them.
// RemoveSources takes the sources out, and their chunks and vectors with them,
// the way a foreign key does in the database.
func (s *store) RemoveSources(_ context.Context, vaultID string, kind domain.SourceKind, paths []string) error {
	held := s.sources[vaultID]
	for _, path := range paths {
		if src, ok := held[path]; ok && src.Ref.Kind == kind {
			delete(held, path)
			s.clear(vaultID, path)
		}
	}
	return nil
}

// MoveSources files what was at one path, and everything under it, where it now
// is. The chunks travel with the source, as they do in the database.
func (s *store) MoveSources(_ context.Context, vaultID, from, to string) error {
	held := s.sources[vaultID]
	for path, src := range held {
		if path != from && !strings.HasPrefix(path, from+"/") {
			continue
		}
		landed := to + path[len(from):]
		delete(held, path)
		src.Ref.Path = landed
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
func (s *store) Under(_ context.Context, vaultID, path string) ([]domain.Fingerprint, error) {
	var out []domain.Fingerprint
	for held, src := range s.sources[vaultID] {
		if held == path || strings.HasPrefix(held, path+"/") {
			out = append(out, src.Ref)
		}
	}
	slices.SortFunc(out, func(a, b domain.Fingerprint) int { return strings.Compare(a.Path, b.Path) })
	return out, nil
}

func (s *store) clear(vaultID, path string) {
	kept := s.chunks[:0]
	for _, c := range s.chunks {
		if c.vault == vaultID && c.path == path {
			delete(s.vectors, c.id)
			continue
		}
		kept = append(kept, c)
	}
	s.chunks = kept
}

func (s *store) insert(vaultID string, ref domain.Fingerprint, c port.Chunk, parent int64) int64 {
	s.next++
	s.chunks = append(s.chunks, storedChunk{
		id: s.next, vault: vaultID, path: ref.Path, kind: ref.Kind,
		start: c.Start, length: c.Length, location: c.Location, text: c.Text, parent: parent,
	})
	return s.next
}

// paths answers a question about sources as the paths that satisfy it, ordered
// and bounded the way the index orders and bounds them.
func (s *store) paths(vaultID string, kind domain.SourceKind, limit int, owing func(port.Source) bool) ([]string, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("a question about sources needs a positive limit, got %d", limit)
	}
	var out []string
	for _, src := range s.sources[vaultID] {
		if src.Ref.Kind == kind && owing(src) {
			out = append(out, src.Ref.Path)
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
func (s *store) cut(vaultID, path string) bool {
	for _, c := range s.chunks {
		if c.vault == vaultID && c.path == path && c.parent != 0 {
			return true
		}
	}
	return false
}

// holds says whether the index holds a chunk of that number.
func (s *store) holds(chunk int64) bool {
	for _, c := range s.chunks {
		if c.id == chunk {
			return true
		}
	}
	return false
}

func (s *store) embedded(chunk int64, model port.EmbeddingModel) bool {
	for _, v := range s.vectors[chunk] {
		if v.Model == model {
			return true
		}
	}
	return false
}

// ordered is the chunks by their own number, which is the order every answer
// about them is given in.
func (s *store) ordered() []storedChunk {
	out := slices.Clone(s.chunks)
	slices.SortFunc(out, func(a, b storedChunk) int { return cmp.Compare(a.id, b.id) })
	return out
}

// small is the chunks of one vault that carry a vector: the chunks a search
// runs over.
func (s *store) small(vaultID string) []storedChunk {
	var out []storedChunk
	for _, c := range s.ordered() {
		if c.vault == vaultID && c.parent != 0 {
			out = append(out, c)
		}
	}
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
	kind  domain.SourceKind
	raw   []byte
	mtime int64
}

func newLibrary() *library {
	return &library{files: map[string]*shelved{}, reads: map[string]int{}}
}

// hold puts one file in the vault, at the modification time given.
func (l *library) hold(path string, kind domain.SourceKind, raw []byte, mtime int64) {
	l.files[path] = &shelved{kind: kind, raw: raw, mtime: mtime}
}

func (l *library) Walk(_ context.Context, fn func(domain.Fingerprint) error) error {
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

func (l *library) Stat(_ context.Context, path string) (domain.Fingerprint, error) {
	if _, ok := l.files[path]; !ok {
		return domain.Fingerprint{}, fmt.Errorf("stat %s: %w", path, fs.ErrNotExist)
	}
	return l.ref(path), nil
}

func (l *library) ref(path string) domain.Fingerprint {
	held := l.files[path]
	return domain.Fingerprint{Path: path, Kind: held.kind, Size: int64(len(held.raw)), MTime: held.mtime}
}

// vaults opens the reader of each vault a test set up.
type vaults map[string]*library

func (v vaults) Open(vault domain.Vault) (port.VaultReader, error) {
	reader, ok := v[string(vault.ID)]
	if !ok {
		return nil, fmt.Errorf("no vault %s", string(vault.ID))
	}
	return reader, nil
}

// embedder is a model that answers the same vector for the same text, so that
// what it was asked is visible in what it returned.
//
// `refuse` is the call at which it fails, standing for a network that went away
// part way through a run.
type embedder struct {
	dims   int
	refuse int

	calls int
	seen  []string // every text handed over, in the order it arrived
}

func (e *embedder) Model() port.EmbeddingModel {
	return port.EmbeddingModel{Name: "fake", Dimensions: e.dims}
}

func (*embedder) Close() error { return nil }

func (e *embedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	e.calls++
	if e.refuse > 0 && e.calls == e.refuse {
		return nil, fmt.Errorf("the model is not there")
	}
	out := make([][]float32, 0, len(texts))
	for _, text := range texts {
		e.seen = append(e.seen, text)
		out = append(out, vectorOf(text, e.dims))
	}
	return out, nil
}

// vectorOf is a direction that belongs to one text and no other.
func vectorOf(text string, dims int) []float32 {
	sum := fnv.New64a()
	sum.Write([]byte(text))
	numbers := rand.New(rand.NewPCG(sum.Sum64(), 0x9E3779B97F4A7C15))

	out := make([]float32, dims)
	for i := range out {
		out[i] = float32(numbers.NormFloat64())
	}
	return out
}

// bookOf is the bytes of an EPUB whose spine is the parts given, each under a
// heading of its own, so that the book names its places.
func bookOf(t *testing.T, title string, parts ...string) []byte {
	t.Helper()

	var buf bytes.Buffer
	archive := zip.NewWriter(&buf)
	add := func(name, body string) {
		file, err := archive.Create(name)
		if err != nil {
			t.Fatalf("write the fixture: %v", err)
		}
		if _, err := file.Write([]byte(body)); err != nil {
			t.Fatalf("write the fixture: %v", err)
		}
	}

	add("mimetype", "application/epub+zip")
	add("META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)

	var manifest, spine strings.Builder
	for i, part := range parts {
		name := fmt.Sprintf("part%d.xhtml", i+1)
		fmt.Fprintf(&manifest, `<item id="p%d" href="%s" media-type="application/xhtml+xml"/>`, i+1, name)
		fmt.Fprintf(&spine, `<itemref idref="p%d"/>`, i+1)
		add("OEBPS/"+name, fmt.Sprintf(
			`<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>%[1]s</title></head>`+
				`<body><h1>Part %[2]d</h1><p>%[3]s</p></body></html>`, title, i+1, part))
	}
	add("OEBPS/content.opf", fmt.Sprintf(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>%[1]s</dc:title>
    <dc:identifier id="id">%[1]s</dc:identifier>
  </metadata>
  <manifest>%[2]s</manifest>
  <spine>%[3]s</spine>
</package>`, title, manifest.String(), spine.String()))

	if err := archive.Close(); err != nil {
		t.Fatalf("write the fixture: %v", err)
	}
	return buf.Bytes()
}

// words is n words of one vocabulary. A vault's books are written in its own, so
// a chunk says which vault it came from.
func words(vocabulary []string, n int) string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, vocabulary[i%len(vocabulary)])
	}
	return strings.Join(out, " ")
}

func (s *store) Reading(_ context.Context, vaultID, path string) (port.SourceText, bool, error) {
	src, held := s.sources[vaultID][path]
	if !held {
		return port.SourceText{}, false, nil
	}
	// The row says what the file was when it was read, as the query does.
	return port.SourceText{
		Path: path, Producer: src.TextFrom, Hash: src.Hash,
		Size: src.Ref.Size, MTime: src.Ref.MTime,
	}, true, nil
}

func (s *store) Recognised(_ context.Context, vaultID string, kind domain.SourceKind) ([]port.SourceText, error) {
	var out []port.SourceText
	for path, src := range s.sources[vaultID] {
		if src.Ref.Kind == kind && src.TextFrom != "" {
			out = append(out, port.SourceText{Path: path, Producer: src.TextFrom, Hash: src.Hash})
		}
	}
	slices.SortFunc(out, func(a, b port.SourceText) int { return cmp.Compare(a.Path, b.Path) })
	return out, nil
}
