package searchquery_test

import (
	"context"
	"encoding/hex"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/celquery"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/sourcefake"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery/searchquerytest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

const (
	dimensions = 1024
	testFloor  = 0.3
)

// world is one vault of invented files, scanned into a real index.
type world struct {
	db *index.DB
	v  domain.Vault
}

func newWorld(t *testing.T, files map[string]string) world {
	t.Helper()
	return newWorldAt(t, files, nil)
}

func newWorldAt(t *testing.T, files map[string]string, modified map[string]time.Time) world {
	t.Helper()
	db, err := index.Open(t.Context(), indexfile.GetPath(t), index.DisableSync())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	v := vaulttest.NewVault(t, files)
	for name, instant := range modified {
		if err := os.Chtimes(filepath.Join(v.Path, filepath.FromSlash(name)), instant, instant); err != nil {
			t.Fatal(err)
		}
	}
	scan := indexing.NewScan(vaultreader.Readers{}, db.Vaults(), db.Notes(), db.NoteQueries(), db.Maintenance(), testsupport.EmptySourceBacklog{})
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	return world{db: db, v: v}
}

func (w world) read(t *testing.T, filePath string) string {
	t.Helper()
	reader, err := vaultreader.Readers{}.Open(w.v)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := reader.Read(t.Context(), filePath)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func (w world) cutBySize(t *testing.T, filePath string, size int) {
	t.Helper()
	text := w.read(t, filePath)
	whole := chunk.Chunk{Start: 0, Length: len(text), Text: text}
	for start := 0; start < len(text); {
		end := min(start+size, len(text))
		for end < len(text) && text[end] != ' ' {
			end++
		}
		whole.Small = append(whole.Small, chunk.Chunk{Start: start, Length: end - start, Text: text[start:end]})
		start = end
	}
	w.replace(t, filePath, []chunk.Chunk{whole})
}

func (w world) cutRanges(t *testing.T, filePath string, ranges ...[2]int) {
	t.Helper()
	text := w.read(t, filePath)
	var chunks []chunk.Chunk
	for _, r := range ranges {
		part := text[r[0]:r[1]]
		small := chunk.Chunk{Start: r[0], Length: r[1] - r[0], Text: part}
		chunks = append(chunks, chunk.Chunk{Start: r[0], Length: r[1] - r[0], Text: part, Small: []chunk.Chunk{small}})
	}
	w.replace(t, filePath, chunks)
}

func (w world) replace(t *testing.T, filePath string, chunks []chunk.Chunk) {
	t.Helper()
	if err := w.db.Chunks().ReplaceChunks(t.Context(), w.v.ID, "note", filePath, chunks); err != nil {
		t.Fatal(err)
	}
}

func (w world) vectorize(t *testing.T, filePath string, direction func(record chunk.Record) []float32) {
	t.Helper()
	unembedded, err := w.db.ChunkVectors().GetUnembeddedChunks(t.Context(), w.v.ID, getModel().Recipe(), chunk.Cursor{}, 5000)
	if err != nil {
		t.Fatal(err)
	}
	var vectors []chunk.Vector
	for _, p := range unembedded {
		if p.Path != filePath {
			continue
		}
		raw, err := hex.DecodeString(p.ChunkHash)
		if err != nil {
			t.Fatal(err)
		}
		one := direction(p)
		vectors = append(vectors, chunk.Vector{Chunk: p.Chunk, Hash: raw, Recipe: getModel().Recipe(), Value: precise(one), Coarse: vector.QuantizeBits(one)})
	}
	if len(vectors) == 0 {
		t.Fatalf("no small chunk of %s to embed", filePath)
	}
	if err := w.db.Chunks().SaveVectors(t.Context(), vectors); err != nil {
		t.Fatal(err)
	}
}

func precise(v []float32) []byte {
	q := vector.QuantizeBytes(vector.Normalize(slices.Clone(v)))
	out := make([]byte, len(q))
	for i, x := range q {
		out[i] = byte(x)
	}
	return out
}

func getModel() searchsource.Description {
	return searchsource.Description{Name: "test", Width: dimensions}
}

func newVector(flipped int) []float32 {
	v := make([]float32, dimensions)
	for i := range v {
		v[i] = 1
		if i < flipped {
			v[i] = -1
		}
	}
	return v
}

// texts is a model that embeds each query with the vector of its text.
type texts map[string][]float32

func (texts) Describe() searchsource.Description { return getModel() }

func (m texts) Embed(_ context.Context, queries []string) ([][]float32, error) {
	out := make([][]float32, len(queries))
	for i, q := range queries {
		out[i] = m[q]
	}
	return out, nil
}

type fixedFloor float64

func (f fixedFloor) GetFloor(context.Context, domain.Vault, string) float64 { return float64(f) }

// extensionTypes is a content type table by the extension of a file.
type extensionTypes struct{}

func (extensionTypes) GetContentType(name string) domain.ContentType {
	switch path.Ext(name) {
	case ".pdf":
		return "application/pdf"
	case ".md":
		return "text/markdown"
	}
	return domain.UnknownContentType
}

func (extensionTypes) ListExtensions(domain.ContentType) []string { return nil }

func (w world) newFind(model texts) *searchquery.FindSearchResults {
	return searchquerytest.New(w.newDeps(model))
}

// countedVectors counts the approximate searches it passes to the index.
type countedVectors struct {
	searchquery.VectorQueries
	coarseSearches int
}

func (c *countedVectors) FindCoarseCandidates(ctx context.Context, vaultID domain.VaultID, q source.CoarseQuery) ([]domain.ChunkID, error) {
	c.coarseSearches++
	return c.VectorQueries.FindCoarseCandidates(ctx, vaultID, q)
}

func (w world) newCountedFind(model texts) (*searchquery.FindSearchResults, *countedVectors) {
	deps := w.newDeps(model)
	counted := &countedVectors{VectorQueries: deps.Evaluate.Vectors}
	deps.Evaluate.Vectors = counted
	return searchquerytest.New(deps), counted
}

func (w world) newDeps(model texts) searchquerytest.Deps {
	search := w.db.ChunkSearch()
	deps := searchquerytest.Deps{
		Parser: celquery.NewParser(searchquerydomain.DefaultLimits()), Clock: newClock(),
		Evaluate: searchquery.EvaluateQueryDeps{
			Documents: search, Vectors: search, Places: search, Contents: w.db.NoteQueryReads(), Links: w.db.NoteQueryReads(),
			Matcher: wordmatch.NewMatcher(),
			Readers: vaultreader.Readers{}, Floors: fixedFloor(testFloor), ContentTypes: extensionTypes{},
		},
	}
	if model != nil {
		deps.Evaluate.Source = sourcefake.Wrap(model)
	}
	return deps
}

func getPaths(results searchquerydomain.SearchResults) []string {
	out := make([]string, len(results.Results))
	for i, r := range results.Results {
		out[i] = r.Document.Path
	}
	return out
}

func repeatSentences(word string, count int) string {
	var b strings.Builder
	for i := range count {
		b.WriteString(word)
		b.WriteString(" sentence number ")
		b.WriteString(strings.Repeat("x", i%7+1))
		b.WriteString(" of the filler. ")
	}
	return b.String()
}

func getMoscow() *time.Location {
	return time.FixedZone("Europe/Moscow", 3*60*60)
}

func newClock() *testsupport.ManualClock {
	return testsupport.NewClockAt(time.Date(2026, 11, 15, 12, 0, 0, 0, getMoscow()))
}
