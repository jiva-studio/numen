package search_test

import (
	"context"
	"encoding/hex"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
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

	// A note whose prose stands under a frontmatter block, with the words
	// looked for on the seventh line of that prose.
	isotherm = "---\nid: 01M0DEM0000000000000000002\n---\n" +
		"# Isotherm\n\nA curve of one temperature.\n\n## Below\n\nThe curve holds throughout.\n"
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

	readers := filesystem.VaultReaders{}
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

// cut replaces one note's chunks with a large one and the small ones named,
// each carrying its own words. A small chunk is what a vector belongs to.
func (c corpus) cut(t *testing.T, v domain.Vault, path string, small ...string) {
	t.Helper()
	reader, err := filesystem.VaultReaders{}.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := reader.Read(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	chunks := chunk.Chunk{Start: 0, Length: len(raw), Text: string(raw)}
	for _, text := range small {
		chunks.Small = append(chunks.Small, chunk.Chunk{Start: 0, Length: len(raw), Text: text})
	}
	if err := c.db.Chunks().SaveChunks(t.Context(), string(v.ID), "note", path, []chunk.Chunk{chunks}); err != nil {
		t.Fatal(err)
	}
}

// vectorise gives every small chunk of one vault a vector pointing the way
// given, in both of the representations a chunk carries.
func (c corpus) vectorise(t *testing.T, v domain.Vault, direction []float32) {
	t.Helper()
	owing, err := c.db.ChunkQueries().Unembedded(t.Context(), string(v.ID), model.Recipe(), 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(owing) == 0 {
		t.Fatal("nothing to embed, so a dense half would answer nothing either way")
	}
	vectors := make([]chunk.Vector, 0, len(owing))
	for _, p := range owing {
		raw, err := hex.DecodeString(p.Fingerprint)
		if err != nil {
			t.Fatal(err)
		}
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Fingerprint: raw, Recipe: model.Recipe(),
			Value: precise(direction), Coarse: embedding.Bits(direction),
		})
	}
	if err := c.db.Chunks().SaveVectors(t.Context(), vectors); err != nil {
		t.Fatal(err)
	}
}

// precise is the full-precision half of a vector, one signed byte per dimension.
func precise(v []float32) []byte {
	q := embedding.Bytes(embedding.Normalise(slices.Clone(v)))
	out := make([]byte, len(q))
	for i, x := range q {
		out[i] = byte(x)
	}
	return out
}

func (c corpus) search(embedder port.Embedder) search.Search {
	return search.New(c.db.ChunkQueries(), filesystem.VaultReaders{}, nil, nil, embedder, 0, nil)
}

var model = port.EmbeddingModel{Name: "test", Dimensions: dimensions}

// pointing is a vector whose every dimension has the sign given.
func pointing(sign float32) []float32 {
	v := make([]float32, dimensions)
	for i := range v {
		v[i] = sign
	}
	return v
}

// leaning agrees with `pointing(+1)` on all but an eighth of its dimensions.
// The two stand at a cosine of 0.75, which is near enough for either to be an
// answer to the other and far enough for the nearer one to be recognisable.
func leaning() []float32 {
	v := pointing(+1)
	for i := range dimensions / 8 {
		v[i] = -1
	}
	return v
}

// oneWay is an embedder whose vectors all point the same way.
type oneWay struct{ direction []float32 }

func (o oneWay) Model() port.EmbeddingModel { return model }

func (o oneWay) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for range texts {
		out = append(out, slices.Clone(o.direction))
	}
	return out, nil
}

func (oneWay) Close() error { return nil }

// outOfReach is an embedder on the other side of a network that is not there.
type outOfReach struct{ why error }

func (outOfReach) Model() port.EmbeddingModel { return model }

func (o outOfReach) Embed(context.Context, []string) ([][]float32, error) { return nil, o.why }

func (outOfReach) Close() error { return nil }

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

	// The vaults point different ways, and the query below is the second's own
	// vector: nearest to everything it holds, and near enough to the first's for
	// those to be answers too.
	c.vectorise(t, c.first, leaning())
	c.vectorise(t, c.second, pointing(+1))

	// A search by meaning answers with the whole table's best k, so this is where a
	// lost filter shows.
	dense := c.db.ChunkQueries()
	near, err := dense.Nearest(ctx, string(c.first.ID), model.Recipe(), pointing(+1), nil, 20, search.DefaultFloor)
	if err != nil {
		t.Fatal(err)
	}
	if len(near) == 0 {
		t.Fatal("the first vault's dense half answered nothing at all")
	}
	// Every note of the first vault is a fair answer; nothing of the second is.
	for _, p := range near {
		if _, held := firstNotes[p.Source]; !held {
			t.Errorf("the dense half answered with %s, which the first vault does not hold", p.Source)
		}
	}

	// A full-text match runs across the whole table, and "shared" is in both
	// vaults, so this is where a lost filter shows for a search by words.
	words, err := dense.Lexical(ctx, string(c.second.ID), "shared", nil, 20, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) == 0 {
		t.Fatal("the second vault's words half answered nothing at all")
	}
	for _, p := range words {
		if !strings.Contains(p.Source, "Quasar") {
			t.Errorf("a search by words answered with %s, which is not the second vault's", p.Source)
		}
	}

	// And the merge of the two, which is what a caller asks for.
	found, err := c.search(oneWay{direction: pointing(+1)}).Execute(ctx, c.first, "shared", search.Parameters{})
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
	// Asked from the other side, with the vector the first vault's own chunks
	// carry.
	for _, p := range c.searchIn(t, c.second, "disorder", oneWay{direction: leaning()}) {
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
		"with an embedder and no vector": oneWay{direction: pointing(+1)},
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

func TestAModelOutOfReachLeavesTheWordsToAnswer(t *testing.T) {
	ctx := t.Context()
	c := indexed(t)

	var said []error
	finds := search.New(c.db.ChunkQueries(), filesystem.VaultReaders{}, nil, nil,
		outOfReach{why: errors.New("dial tcp: network is unreachable")}, 0,
		func(err error) { said = append(said, err) })

	found, err := finds.Execute(ctx, c.first, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || !strings.Contains(found[0].Source, "Entropy") {
		t.Fatalf("the words alone found %v", sources(found))
	}
	if len(said) != 1 {
		t.Fatalf("the half that did not answer said %v", said)
	}
}

func TestASearchTheCallerStoppedIsNotAnAnswer(t *testing.T) {
	// A question nobody is waiting for any more is not a question the words
	// answer on their own.
	ctx := t.Context()
	c := indexed(t)

	finds := search.New(c.db.ChunkQueries(), filesystem.VaultReaders{}, nil, nil,
		outOfReach{why: context.Canceled}, 0, func(error) {})

	if _, err := finds.Execute(ctx, c.first, "disorder", search.Parameters{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestFiveMatchingChunksOfOneNoteAreOneResult(t *testing.T) {
	ctx := t.Context()
	c := indexed(t)
	c.cut(t, c.first, "notes/Entropy.md",
		"disorder once", "disorder twice", "disorder again",
		"disorder once more", "disorder at last")

	// Every chunk of the note matches, the large one included.
	hits, err := c.db.ChunkQueries().Lexical(ctx, string(c.first.ID), "disorder", nil, 20, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 6 {
		t.Fatalf("%d chunks of the note match, want its five small chunks and the large one", len(hits))
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

// sectioned replaces one note's chunks with three, of which the first opens a
// section whose name the section after it says over and over.
//
// This is the shape a chapter of a book has: the chapter names its subject once,
// in its heading, and a paragraph further on says it four times.
func (c corpus) sectioned(t *testing.T, v domain.Vault, path string) {
	t.Helper()
	chunks := []chunk.Chunk{
		{
			Start: 0, Length: 40, Location: "Madhavendra Puri",
			Opens: []string{"Madhavendra Puri"},
			Text:  "Madhavendra Puri appeared in the fourteenth",
		},
		{
			Start: 40, Length: 40, Location: "The Disciplic Succession",
			Opens: []string{"The Disciplic Succession"},
			Text: "Madhavendra Puri was the disciple. Madhavendra Puri had disciples. " +
				"Madhavendra Puri is said. Madhavendra Puri again.",
		},
	}
	if err := c.db.Chunks().SaveChunks(t.Context(), string(v.ID), "note", path, chunks); err != nil {
		t.Fatal(err)
	}
}

func TestASearchAnswersWithTheSectionAskedAbout(t *testing.T) {
	// A search by words ranks by how often the words appear, so a paragraph in the
	// middle of a chapter outranks the chapter's own opening. Asked where a book
	// speaks about a thing, what a person wants is the section about it.
	ctx := t.Context()
	c := indexed(t)
	c.sectioned(t, c.first, "notes/Entropy.md")

	words, err := c.db.ChunkQueries().Lexical(ctx, string(c.first.ID), "Madhavendra Puri", nil, 20, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) == 0 || words[0].Start == 0 {
		t.Fatalf("a search by words answers with %+v, and this proves nothing", words)
	}

	found, err := c.search(nil).Execute(ctx, c.first, "Madhavendra Puri", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatal("the search found nothing")
	}
	if found[0].Start != 0 {
		t.Errorf("the search opened at %d, and the section begins at 0", found[0].Start)
	}
	if found[0].Location != "Madhavendra Puri" {
		t.Errorf("the search answered with %q", found[0].Location)
	}
}

func TestAPassageCarriesTheLineItStandsOnInTheProse(t *testing.T) {
	// The window puts the caret where the words were found. A note's
	// frontmatter stands before its prose and is no line of it.
	ctx := t.Context()
	c := indexed(t)

	const path = "notes/Isotherm.md"
	const held = "The curve holds throughout."
	v := testsupport.NewVault(t, map[string]string{path: isotherm})
	scan := usecase.Scan{
		Readers: filesystem.VaultReaders{}, Vaults: c.db.Vaults(), Notes: c.db.Notes(),
		Known: c.db.NoteQueries(), Maintenance: c.db.Statistics(),
	}
	if _, err := scan.Execute(ctx, v); err != nil {
		t.Fatal(err)
	}

	at := strings.Index(isotherm, held)
	whole := chunk.Chunk{
		Start: 0, Length: len(isotherm), Text: isotherm,
		Small: []chunk.Chunk{{Start: at, Length: len(held), Text: held}},
	}
	if err := c.db.Chunks().SaveChunks(ctx, string(v.ID), "note", path, []chunk.Chunk{whole}); err != nil {
		t.Fatal(err)
	}

	found := c.searchIn(t, v, "throughout", nil)
	if len(found) != 1 {
		t.Fatalf("want the one note holding the word: %v", sources(found))
	}
	if found[0].Line != 6 {
		t.Errorf("the passage stands on line %d, and the words are on the seventh line of the prose",
			found[0].Line)
	}
}

func TestANameThatMatchesNoSectionChangesNothing(t *testing.T) {
	// A search by name answers about sections alone. A question about words that
	// name no section is the search there was before it.
	ctx := t.Context()
	c := indexed(t)
	c.sectioned(t, c.first, "notes/Entropy.md")

	found, err := c.search(nil).Execute(ctx, c.first, "disciple", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatal("the search found nothing")
	}
	if found[0].Start != 40 {
		t.Errorf("the search opened at %d, and the words are at 40", found[0].Start)
	}
}

func TestASearchByMeaningAnswersWhereWordsCannot(t *testing.T) {
	// A vector is kept under the recipe it was made by, which is everything
	// about the model that decides what a vector is. Asked under anything else
	// — a name, a name and a width — no vector is found and this half answers
	// nothing at all, silently, for ever.
	//
	// Every other test here has a query a search by words can answer, so a meaning
	// half that answered nothing was a search that still looked right.
	ctx := t.Context()
	c := indexed(t)
	c.vectorise(t, c.first, pointing(+1))

	// A word no note in the vault says, so nothing lexical can match it.
	const unsaid = "zzqqxx"
	words, err := c.db.ChunkQueries().Lexical(ctx, string(c.first.ID), unsaid, nil, 20, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 0 {
		t.Fatalf("a search by words answered %d passages to a word nothing says", len(words))
	}

	found, err := c.search(oneWay{pointing(+1)}).Execute(ctx, c.first, unsaid, search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatal("a search by meaning answered nothing, so the search is its words alone")
	}
}

func TestASearchByMeaningAsksUnderTheRecipeAVectorIsKeptBy(t *testing.T) {
	// The recipe is what the vector was written under. A search asking under
	// anything else joins on nothing, and the failure is silence rather than an
	// error: a search by words answers and the search looks like it worked.
	ctx := t.Context()
	c := indexed(t)
	c.vectorise(t, c.first, pointing(+1))

	under, err := c.db.ChunkQueries().Nearest(
		ctx, string(c.first.ID), model.Recipe(), pointing(+1), nil, 10, search.DefaultFloor)
	if err != nil {
		t.Fatal(err)
	}
	if len(under) == 0 {
		t.Fatal("nothing came back under the recipe the vectors were written with")
	}

	astray, err := c.db.ChunkQueries().Nearest(
		ctx, string(c.first.ID), model.String(), pointing(+1), nil, 10, search.DefaultFloor)
	if err != nil {
		t.Fatal(err)
	}
	if len(astray) != 0 {
		t.Errorf("%d passages came back under a name that is not the recipe", len(astray))
	}
}

func TestAQuestionAboutBooksIsAnsweredFromBooks(t *testing.T) {
	// A vault holds far more notes than books. Asked about a book and told to
	// look everywhere, a search answers with whatever the vault holds most of,
	// and the person who said "in the book" is handed a note.
	ctx := t.Context()
	c := indexed(t)
	c.sectioned(t, c.first, "notes/Entropy.md")
	c.vectorise(t, c.first, pointing(+1))

	everywhere, err := c.search(oneWay{pointing(+1)}).
		Execute(ctx, c.first, "Madhavendra Puri", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(everywhere) == 0 {
		t.Fatal("the search found nothing at all")
	}

	// This vault holds notes alone, so a question about books is answered by
	// nothing: what is asserted is that the kind reached every half.
	books, err := c.search(oneWay{pointing(+1)}).Execute(ctx, c.first, "Madhavendra Puri",
		search.Parameters{Of: []domain.SourceKind{domain.KindBook}})
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 0 {
		t.Errorf("%d passages came back from the books of a vault that holds none", len(books))
	}

	notes, err := c.search(oneWay{pointing(+1)}).Execute(ctx, c.first, "Madhavendra Puri",
		search.Parameters{Of: []domain.SourceKind{domain.KindNote}})
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != len(everywhere) {
		t.Errorf("asking the notes of a vault of notes gave %d passages and asking everywhere gave %d",
			len(notes), len(everywhere))
	}
}

func TestEveryWayIsToldWhichKindsAQuestionIsAbout(t *testing.T) {
	// One half left unfiltered answers about the wrong kind, and the fused
	// order carries it: a way that ignores the kind is a half that undoes it.
	ctx := t.Context()
	c := indexed(t)
	c.sectioned(t, c.first, "notes/Entropy.md")
	c.vectorise(t, c.first, pointing(+1))
	queries := c.db.ChunkQueries()
	books := []domain.SourceKind{domain.KindBook}

	words, err := queries.Lexical(ctx, string(c.first.ID), "Madhavendra Puri", books, 20, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 0 {
		t.Errorf("a search by words answered %d passages about books in a vault of notes", len(words))
	}

	named, err := queries.Named(ctx, string(c.first.ID), "Madhavendra Puri", books, 20, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(named) != 0 {
		t.Errorf("a search by name answered %d sections of books in a vault of notes", len(named))
	}

	dense, err := queries.Nearest(ctx, string(c.first.ID), model.Recipe(), pointing(+1), books, 20, -1)
	if err != nil {
		t.Fatal(err)
	}
	if len(dense) != 0 {
		t.Errorf("a search by meaning answered %d passages of books in a vault of notes", len(dense))
	}
}
