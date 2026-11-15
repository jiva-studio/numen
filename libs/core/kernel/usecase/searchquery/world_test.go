package searchquery_test

import (
	"context"
	"encoding/hex"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/celquery"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	searchquerydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/sourcefake"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
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
	db, err := index.Open(t.Context(), indexfile.Path(t), index.Unsynchronized())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	v := vaultfixture.NewVault(t, files)
	scan := vaults.NewScan(vaultreader.Readers{}, db.Vaults(), db.Notes(), db.NoteQueries(), db.Maintenance(), none.SourceBacklog{})
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

// cutBySize cuts a note into one large chunk and small chunks of about size bytes.
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

// cutRanges cuts a note into large chunks at the ranges, each with one small chunk of the same range.
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
		vectors = append(vectors, chunk.Vector{Chunk: p.Chunk, Hash: raw, Recipe: getModel().Recipe(), Value: precise(one), Coarse: vector.Bits(one)})
	}
	if len(vectors) == 0 {
		t.Fatalf("no small chunk of %s to embed", filePath)
	}
	if err := w.db.Chunks().SaveVectors(t.Context(), vectors); err != nil {
		t.Fatal(err)
	}
}

func precise(v []float32) []byte {
	q := vector.Bytes(vector.Normalize(slices.Clone(v)))
	out := make([]byte, len(q))
	for i, x := range q {
		out[i] = byte(x)
	}
	return out
}

func getModel() searchsource.Description {
	return searchsource.Description{Name: "test", Width: dimensions}
}

// newVector returns a vector of ones with the first `flipped` values negated: its similarity with
// newVector(0) is 1 - 2*flipped/dimensions.
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
	search := w.db.ChunkSearch()
	deps := searchquery.FindSearchResultsDeps{
		Parser: celquery.NewParser(searchquerydomain.DefaultLimits()), Documents: search, Vectors: search, Words: search,
		Places: search, Matcher: wordmatch.NewMatcher(), Readers: vaultreader.Readers{},
		Floors: fixedFloor(testFloor), ContentTypes: extensionTypes{},
	}
	if model != nil {
		deps.Source = sourcefake.Wrap(model)
	}
	return searchquery.NewFindSearchResults(deps)
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
