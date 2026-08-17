package search_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/embedding"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/search"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// dimensions is the width the vector index is built at.
const dimensions = 1024

// The two vaults every test here uses. They share one word and nothing else, so
// a result from the wrong one is recognisable.
var (
	firstNotes = map[string]string{
		"notes/Entropy.md": "# Entropy\n\nA measure of disorder, shared by every closed system.\n",
		"notes/Heat.md":    "# Heat\n\nEnergy in transit, shared between two bodies.\n",
	}
	secondNotes = map[string]string{
		"notes/Quasar.md": "# Quasar\n\nA distant beacon, shared by no textbook here.\n",
	}
)

// corpus is two vaults, indexed and cut, and the one search over them.
type corpus struct {
	db     *index.DB
	first  domain.Vault
	second domain.Vault
}

func indexed(t *testing.T) corpus {
	t.Helper()
	ctx := t.Context()

	db, err := index.Open(ctx, filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	readers := filesystem.Readers{}
	scan := usecase.Scan{
		Readers: readers, Vaults: db.Vaults(), Notes: db.Notes(),
		Known: db.NoteQueries(), Maintenance: db.Statistics(),
	}

	c := corpus{db: db}
	for i, notes := range []map[string]string{firstNotes, secondNotes} {
		v := testsupport.NewVault(t, notes)
		if _, err := scan.Execute(ctx, v); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			c.first = v
		} else {
			c.second = v
		}
	}
	return c
}

// cut replaces one note's windows with a large one and the small ones named,
// each carrying its own words. A small window is what a vector belongs to.
func (c corpus) cut(t *testing.T, v domain.Vault, path string, small ...string) {
	t.Helper()
	reader, err := filesystem.Readers{}.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := reader.Read(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	windows := chunk.Window{Start: 0, Length: len(raw), Text: string(raw)}
	for _, text := range small {
		windows.Small = append(windows.Small, chunk.Window{Start: 0, Length: len(raw), Text: text})
	}
	if err := c.db.Chunks().SaveWindows(t.Context(), v.ID, "note", path, []chunk.Window{windows}); err != nil {
		t.Fatal(err)
	}
}

// vectorise gives every small window of one vault a vector pointing the same
// way, so a query pointing the other way is as far from it as the space allows.
func (c corpus) vectorise(t *testing.T, v domain.Vault, sign float32) {
	t.Helper()
	owing, err := c.db.ChunkQueries().Unembedded(t.Context(), v.ID, model.String(), 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) == 0 {
		t.Fatal("nothing to embed, so a dense half would answer nothing either way")
	}
	vectors := make([]chunk.Vector, 0, len(owing))
	for _, p := range owing {
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Model: model.String(), Dims: dimensions, Kind: "int8",
			Value: []byte{}, Coarse: embedding.Bits(pointing(sign)),
		})
	}
	if err := c.db.Chunks().SaveVectors(t.Context(), vectors); err != nil {
		t.Fatal(err)
	}
}

func (c corpus) search(embedder port.Embedder) search.Search {
	return search.New(c.db.ChunkQueries(), filesystem.Readers{}, embedder)
}

var model = port.EmbeddingModel{Name: "test", Dimensions: dimensions}

// pointing is a vector whose every dimension has the sign given, which is the
// whole of what the coarse pass reads.
func pointing(sign float32) []float32 {
	v := make([]float32, dimensions)
	for i := range v {
		v[i] = sign
	}
	return v
}

// oneWay is an embedder whose vectors all point the same way.
type oneWay struct{ sign float32 }

func (o oneWay) Model() port.EmbeddingModel { return model }

func (o oneWay) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for range texts {
		out = append(out, pointing(o.sign))
	}
	return out, nil
}

func sources(passages []domain.Passage) []string {
	out := make([]string, 0, len(passages))
	for _, p := range passages {
		out = append(out, p.Source)
	}
	return out
}

func TestASearchAnswersFromItsOwnVaultAlone(t *testing.T) {
	// One database holds every vault, and each half is checked on its own: either
	// can lose the vault filter.
	ctx := t.Context()
	c := indexed(t)
	c.cut(t, c.first, "notes/Entropy.md", "a measure of disorder")
	c.cut(t, c.second, "notes/Quasar.md", "a distant beacon")

	// The vaults point opposite ways, so the query below is nearest to
	// everything the second vault holds and furthest from the first's.
	c.vectorise(t, c.first, -1)
	c.vectorise(t, c.second, +1)

	// The coarse pass answers with the whole table's best k, so this is where a
	// lost filter shows.
	dense := c.db.ChunkQueries()
	near, err := dense.Nearest(ctx, c.first.ID, embedding.Bits(pointing(+1)), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(near) == 0 {
		t.Fatal("the first vault's dense half answered nothing at all")
	}
	for _, p := range near {
		if !strings.Contains(p.Source, "Entropy") {
			t.Errorf("the dense half answered with %s, which is not the first vault's", p.Source)
		}
	}

	// A full-text match runs across the whole table, and "shared" is in both
	// vaults, so this is where a lost filter shows for the words half.
	words, err := dense.Lexical(ctx, c.second.ID, "shared", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) == 0 {
		t.Fatal("the second vault's words half answered nothing at all")
	}
	for _, p := range words {
		if !strings.Contains(p.Source, "Quasar") {
			t.Errorf("the words half answered with %s, which is not the second vault's", p.Source)
		}
	}

	// And the merge of the two, which is what a caller asks for.
	found, err := c.search(oneWay{sign: +1}).Execute(ctx, c.first, "shared", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatal("the first vault answered nothing at all")
	}
	for _, p := range found {
		if strings.Contains(p.Source, "Quasar") || strings.Contains(p.Text, "beacon") {
			t.Errorf("the merge answered with %+v, which belongs to the second vault", p)
		}
	}
	// Asked from the other side, with the vector that points at the first
	// vault's own chunks.
	for _, p := range c.searchIn(t, c.second, "disorder", oneWay{sign: -1}) {
		if !strings.Contains(p.Source, "Quasar") {
			t.Errorf("the second vault answered with %s, which belongs to the first", p.Source)
		}
	}
}

func (c corpus) searchIn(t *testing.T, v domain.Vault, query string, embedder port.Embedder) []domain.Passage {
	t.Helper()
	found, err := c.search(embedder).Execute(t.Context(), v, query, search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func TestAVaultWithNoVectorsAnswersFromItsWords(t *testing.T) {
	// The vector index is optional and may never finish filling, so the words
	// half alone is a whole search.
	ctx := t.Context()
	c := indexed(t)

	for name, embedder := range map[string]port.Embedder{
		"with no embedder at all":        nil,
		"with an embedder and no vector": oneWay{sign: +1},
	} {
		t.Run(name, func(t *testing.T) {
			found, err := c.search(embedder).Execute(ctx, c.first, "disorder", search.Parameters{})
			if err != nil {
				t.Fatal(err)
			}
			if len(found) != 1 || !strings.Contains(found[0].Source, "Entropy") {
				t.Fatalf("the words alone found %v", sources(found))
			}
			if !strings.Contains(found[0].Text, "A measure of disorder") {
				t.Errorf("the passage does not carry the text around the hit: %q", found[0].Text)
			}
		})
	}
}

func TestFiveMatchingChunksOfOneNoteAreOneResult(t *testing.T) {
	ctx := t.Context()
	c := indexed(t)
	c.cut(t, c.first, "notes/Entropy.md",
		"disorder once", "disorder twice", "disorder again",
		"disorder once more", "disorder at last")

	// Every window of the note matches, the large one included.
	hits, err := c.db.ChunkQueries().Lexical(ctx, c.first.ID, "disorder", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 6 {
		t.Fatalf("%d chunks of the note match, want its five small windows and the large one", len(hits))
	}

	found := c.searchIn(t, c.first, "disorder", nil)
	if len(found) != 1 {
		t.Errorf("one note contributing six matching chunks gave %v", sources(found))
	}
}

func TestAPassageIsReadFromTheFileAndNotFromTheIndex(t *testing.T) {
	// The text is not stored. A passage carries the words only because the file
	// was opened at the place the chunk names.
	c := indexed(t)

	found := c.searchIn(t, c.first, "shared", nil)
	if len(found) != 2 {
		t.Fatalf("want a passage from each note that holds the word: %v", sources(found))
	}
	for _, p := range found {
		if p.Text == "" {
			t.Errorf("%s came back with no text", p.Source)
		}
		if strings.Contains(p.Text, "---") {
			t.Errorf("%s came back with the frontmatter in it: %q", p.Source, p.Text)
		}
	}
}
