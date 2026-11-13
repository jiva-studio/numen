package search_test

import (
	"context"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/sourcefake"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// dimensions is the width the vector index is built at.
const dimensions = 1024

func getFirstNotes() map[string]string {
	return map[string]string{
		"notes/Entropy.md": "# Entropy\n\nA measure of disorder, shared by every closed system.\n",
		"notes/Heat.md":    "# Heat\n\nEnergy in transit, shared between two bodies.\n",
	}
}

func getSecondNotes() map[string]string {
	return map[string]string{
		"notes/Quasar.md": "# Quasar\n\nA distant beacon, shared by no textbook here.\n",
	}
}

func getIsotherm() string {
	return "---\nid: 01M0DEM0000000000000000002\n---\n" +
		"# Isotherm\n\nA curve of one temperature.\n\n## Below\n\nThe curve holds throughout.\n"
}

// corpus is two vaults, indexed and cut, and the one search over them.
type corpus struct {
	db     *index.DB
	first  domain.Vault
	second domain.Vault
}

func newCorpus(t *testing.T) corpus {
	t.Helper()
	ctx := t.Context()

	db, err := index.Open(ctx, indexfile.Path(t), index.Unsynchronized())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)

	readers := vaultreader.Readers{}
	scan := vaults.NewScan(
		readers, db.Vaults(), db.Notes(),
		db.NoteQueries(), db.Maintenance(), nil,
	)

	c := corpus{db: db}
	for i, notes := range []map[string]string{getFirstNotes(), getSecondNotes()} {
		v := vaultfixture.NewVault(t, notes)
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

func (c corpus) cut(t *testing.T, v domain.Vault, path string, small ...string) {
	t.Helper()
	reader, err := vaultreader.Readers{}.Open(v)
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
	if err := c.db.Chunks().ReplaceChunks(t.Context(), v.ID, "note", path, []chunk.Chunk{chunks}); err != nil {
		t.Fatal(err)
	}
}

func (c corpus) vectorize(t *testing.T, v domain.Vault, direction []float32) {
	t.Helper()
	unembedded, err := c.db.ChunkVectors().GetUnembeddedChunks(t.Context(), v.ID, getModel().Recipe(), chunk.Cursor{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(unembedded) == 0 {
		t.Fatal("nothing to embed, so a dense half would answer nothing either way")
	}
	vectors := make([]chunk.Vector, 0, len(unembedded))
	for _, p := range unembedded {
		raw, err := hex.DecodeString(p.ChunkHash)
		if err != nil {
			t.Fatal(err)
		}
		vectors = append(vectors, chunk.Vector{
			Chunk: p.Chunk, Hash: raw, Recipe: getModel().Recipe(),
			Value: precise(direction), Coarse: vector.Bits(direction),
		})
	}
	if err := c.db.Chunks().SaveVectors(t.Context(), vectors); err != nil {
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

func (c corpus) participants(model sourcefake.Embedder) []search.Participant {
	if model == nil {
		return nil
	}
	source := sourcefake.Wrap(model).Over(search.NewFindNearest(c.db.ChunkSearch()).Execute)
	return []search.Participant{search.NewSource(source, c.db.ChunkSearch())}
}

func (c corpus) search(model sourcefake.Embedder) search.Search {
	return search.New(c.db.ChunkSearch(), vaultreader.Readers{}, nil, c.participants(model), nil)
}

func getModel() searchsource.Description {
	return searchsource.Description{Name: "test", Width: dimensions}
}

func newVector(sign float32) []float32 {
	v := make([]float32, dimensions)
	for i := range v {
		v[i] = sign
	}
	return v
}

func newNearVector() []float32 {
	v := newVector(+1)
	for i := range dimensions / 8 {
		v[i] = -1
	}
	return v
}

// oneWay is an embedder whose vectors all point the same way.
type oneWay struct{ direction []float32 }

func (o oneWay) Describe() searchsource.Description { return getModel() }

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

func (outOfReach) Describe() searchsource.Description { return getModel() }

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
	c := newCorpus(t)
	c.cut(t, c.first, "notes/Entropy.md", "a measure of disorder, shared")
	c.cut(t, c.second, "notes/Quasar.md", "a distant beacon, shared")

	// The vaults point different ways, and the query below is the second's own
	// vector: nearest to everything it holds, and near enough to the first's for
	// those to be answers too.
	c.vectorize(t, c.first, newNearVector())
	c.vectorize(t, c.second, newVector(+1))

	// A search by meaning answers with the whole table's best k, so this is where a
	// lost filter shows.
	dense := queriesOf(c.db)
	item, err := dense.FindNearest(ctx, c.first.ID, source.NearestQuery{Recipe: getModel().Recipe(), Vector: newVector(+1), Limit: 20, Floor: search.DefaultFloor})
	if err != nil {
		t.Fatal(err)
	}
	if len(item) == 0 {
		t.Fatal("the first vault's dense half answered nothing at all")
	}
	// Every note of the first vault is a fair answer; nothing of the second is.
	for _, p := range item {
		if _, target := getFirstNotes()[p.Source]; !target {
			t.Errorf("the dense half answered with %s, which the first vault does not hold", p.Source)
		}
	}

	// A full-text match runs across the whole table, and "shared" is in both
	// vaults, so this is where a lost filter shows for a search by words.
	words, err := dense.FindLexical(ctx, c.second.ID, source.LexicalQuery{Text: "shared", Kinds: nil, Limit: 20})
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

	// The merged result of both halves.
	found, err := c.search(oneWay{direction: newVector(+1)}).Execute(ctx, c.first, "shared", search.Parameters{})
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
	// Searched from the other side, using the vector the first vault's chunks carry.
	for _, p := range c.searchIn(t, c.second, "disorder", oneWay{direction: newNearVector()}) {
		if !strings.Contains(p.Source, "Quasar") {
			t.Errorf("the second vault answered with %s, which belongs to the first", p.Source)
		}
	}
}

func (c corpus) searchIn(t *testing.T, v domain.Vault, query string, embedder sourcefake.Embedder) []domain.Passage {
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
	c := newCorpus(t)

	for name, embedder := range map[string]sourcefake.Embedder{
		"with no embedder at all":        nil,
		"with an embedder and no vector": oneWay{direction: newVector(+1)},
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

func TestAModelOutOfReachPreservesTheWordsToAnswer(t *testing.T) {
	ctx := t.Context()
	c := newCorpus(t)

	var errs []error
	unreachable := outOfReach{why: errors.New("dial tcp: network is unreachable")}
	finds := search.New(c.db.ChunkSearch(), vaultreader.Readers{}, nil, c.participants(unreachable),
		func(err error) { errs = append(errs, err) })

	found, err := finds.Execute(ctx, c.first, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || !strings.Contains(found[0].Source, "Entropy") {
		t.Fatalf("the words alone found %v", sources(found))
	}
	if len(errs) != 1 {
		t.Fatalf("the half that did not answer said %v", errs)
	}
}

func TestASearchTheCallerStoppedIsNotAnAnswer(t *testing.T) {
	// A search whose caller canceled returns the cancellation error, not a
	// words-only result.
	ctx := t.Context()
	c := newCorpus(t)

	canceled := outOfReach{why: context.Canceled}
	finds := search.New(c.db.ChunkSearch(), vaultreader.Readers{}, nil, c.participants(canceled), func(error) {})

	if _, err := finds.Execute(ctx, c.first, "disorder", search.Parameters{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestFiveMatchingChunksOfOneNoteAreOneResult(t *testing.T) {
	ctx := t.Context()
	c := newCorpus(t)
	c.cut(t, c.first, "notes/Entropy.md",
		"disorder once", "disorder twice", "disorder again",
		"disorder once more", "disorder at last")

	// Every small chunk of the note matches.
	hits, err := c.db.ChunkSearch().FindLexical(ctx, c.first.ID, source.LexicalQuery{Text: "disorder", Kinds: nil, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 5 {
		t.Fatalf("%d chunks of the note match, want its five small chunks", len(hits))
	}

	found := c.searchIn(t, c.first, "disorder", nil)
	if len(found) != 1 {
		t.Errorf("one note contributing five matching chunks gave %v", sources(found))
	}
}

func TestAPassageIsReadFromTheFileAndNotFromTheIndex(t *testing.T) {
	// The text is not stored. A passage carries the words only because the file
	// was opened at the place the chunk names.
	c := newCorpus(t)

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

func (c corpus) writeSections(t *testing.T, v domain.Vault, path string) {
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
	if err := c.db.Chunks().ReplaceChunks(t.Context(), v.ID, "note", path, chunks); err != nil {
		t.Fatal(err)
	}
}

func TestSearch_ReturnsRequestedSection(t *testing.T) {
	// A search by words ranks by how often the words appear, so a paragraph in the
	// middle of a chapter outranks the chapter's own opening.
	ctx := t.Context()
	c := newCorpus(t)
	c.writeSections(t, c.first, "notes/Entropy.md")

	words, err := c.db.ChunkSearch().FindLexical(ctx, c.first.ID, source.LexicalQuery{Text: "Madhavendra Puri", Kinds: nil, Limit: 20})
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

func TestAPassageCarriesTheLineItExistsOnInTheProse(t *testing.T) {
	// The caret is placed where the words were found. A note's frontmatter precedes
	// its prose and is not counted as a line of it.
	ctx := t.Context()
	c := newCorpus(t)

	const path = "notes/Isotherm.md"
	const target = "The curve holds throughout."
	v := vaultfixture.NewVault(t, map[string]string{path: getIsotherm()})
	scan := vaults.NewScan(
		vaultreader.Readers{}, c.db.Vaults(), c.db.Notes(),
		c.db.NoteQueries(), c.db.Maintenance(), nil,
	)
	if _, err := scan.Execute(ctx, v); err != nil {
		t.Fatal(err)
	}

	pos := strings.Index(getIsotherm(), target)
	whole := chunk.Chunk{
		Start: 0, Length: len(getIsotherm()), Text: getIsotherm(),
		Small: []chunk.Chunk{{Start: pos, Length: len(target), Text: target}},
	}
	if err := c.db.Chunks().ReplaceChunks(ctx, v.ID, "note", path, []chunk.Chunk{whole}); err != nil {
		t.Fatal(err)
	}

	found := c.searchIn(t, v, "throughout", nil)
	if len(found) != 1 {
		t.Fatalf("want the one note holding the word: %v", sources(found))
	}
	if found[0].Line != 6 {
		t.Errorf("the passage is on line %d, and the words are on the seventh line of the prose",
			found[0].Line)
	}
}

func TestANameThatMatchesNoSectionChangesNothing(t *testing.T) {
	// A search by name answers about sections alone. A question about words that
	// name no section is the search there was before it.
	ctx := t.Context()
	c := newCorpus(t)
	c.writeSections(t, c.first, "notes/Entropy.md")

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
	// A vector is stored under the recipe that produced it, which covers everything
	// about the model that determines what a vector is. A query under any other recipe
	// (a name, or a name and a width) finds no vector, and this half returns nothing,
	// silently and permanently.
	//
	// Every other test here has a query a search by words can answer, so an empty
	// meaning half would still look like a working search.
	ctx := t.Context()
	c := newCorpus(t)
	c.vectorize(t, c.first, newVector(+1))

	// A word that appears in no note in the vault, so no lexical match is possible.
	const unsaid = "zzqqxx"
	words, err := c.db.ChunkSearch().FindLexical(ctx, c.first.ID, source.LexicalQuery{Text: unsaid, Kinds: nil, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 0 {
		t.Fatalf("a search by words answered %d passages to a word nothing says", len(words))
	}

	found, err := c.search(oneWay{newVector(+1)}).Execute(ctx, c.first, unsaid, search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Fatal("a search by meaning answered nothing, so the search is its words alone")
	}
}

func TestASearchByMeaningAsksBelowTheRecipeAVectorIsCachedBy(t *testing.T) {
	// The recipe is what the vector was written under. A search asking under
	// anything else joins on nothing, and the failure is silence and not an
	// error: a search by words answers and the search looks like it worked.
	ctx := t.Context()
	c := newCorpus(t)
	c.vectorize(t, c.first, newVector(+1))

	limit, err := queriesOf(c.db).FindNearest(ctx, c.first.ID, source.NearestQuery{Recipe: getModel().Recipe(), Vector: newVector(+1), Limit: 10, Floor: search.DefaultFloor})
	if err != nil {
		t.Fatal(err)
	}
	if len(limit) == 0 {
		t.Fatal("nothing came back under the recipe the vectors were written with")
	}

	astray, err := queriesOf(c.db).FindNearest(ctx, c.first.ID, source.NearestQuery{Recipe: getModel().String(), Vector: newVector(+1), Limit: 10, Floor: search.DefaultFloor})
	if err != nil {
		t.Fatal(err)
	}
	if len(astray) != 0 {
		t.Errorf("%d passages came back under a name that is not the recipe", len(astray))
	}
}

func TestSearch_BookQueryUsesBookSources(t *testing.T) {
	// A vault holds far more notes than books. A book query that searches everywhere
	// returns whatever the vault holds most of, so a query scoped to a book returns
	// a note.
	ctx := t.Context()
	c := newCorpus(t)
	c.writeSections(t, c.first, "notes/Entropy.md")
	c.vectorize(t, c.first, newVector(+1))

	everywhere, err := c.search(oneWay{newVector(+1)}).
		Execute(ctx, c.first, "Madhavendra Puri", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(everywhere) == 0 {
		t.Fatal("the search found nothing at all")
	}

	// This vault holds notes alone, so a question about books is answered by
	// nothing: what is asserted is that the kind reached every half.
	books, err := c.search(oneWay{newVector(+1)}).Execute(ctx, c.first, "Madhavendra Puri",
		search.Parameters{Kinds: []domain.SourceKind{domain.KindFile}})
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 0 {
		t.Errorf("%d passages came back from the books of a vault that holds none", len(books))
	}

	notes, err := c.search(oneWay{newVector(+1)}).Execute(ctx, c.first, "Madhavendra Puri",
		search.Parameters{Kinds: []domain.SourceKind{domain.KindNote}})
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != len(everywhere) {
		t.Errorf("asking the notes of a vault of notes gave %d passages and asking everywhere gave %d",
			len(notes), len(everywhere))
	}
}

func TestEveryModeIsReportedWhichKindsAQuestionIsAbout(t *testing.T) {
	// One half left unfiltered answers about the wrong kind, and the fused
	// order carries it: a way that ignores the kind is a half that undoes it.
	ctx := t.Context()
	c := newCorpus(t)
	c.writeSections(t, c.first, "notes/Entropy.md")
	c.vectorize(t, c.first, newVector(+1))
	queries := queriesOf(c.db)
	books := []domain.SourceKind{domain.KindFile}

	words, err := queries.FindLexical(ctx, c.first.ID, source.LexicalQuery{Text: "Madhavendra Puri", Kinds: books, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 0 {
		t.Errorf("a search by words answered %d passages about books in a vault of notes", len(words))
	}

	named, err := queries.GetNamedPassages(ctx, c.first.ID, source.LexicalQuery{Text: "Madhavendra Puri", Kinds: books, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(named) != 0 {
		t.Errorf("a search by name answered %d sections of books in a vault of notes", len(named))
	}

	dense, err := queries.FindNearest(ctx, c.first.ID, source.NearestQuery{Recipe: getModel().Recipe(), Vector: newVector(+1), Kinds: books, Limit: 20, Floor: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(dense) != 0 {
		t.Errorf("a search by meaning answered %d passages of books in a vault of notes", len(dense))
	}
}

func TestASearchAbortsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	c := newCorpus(t)
	searching := c.search(oneWay{direction: newVector(+1)})

	_, err := searching.Execute(ctx, c.first, "disorder", search.Parameters{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// opening is a model that answers no query until its caller stops waiting, as a model that is still being
// opened. asked receives each query it is handed.
type opening struct{ asked chan struct{} }

func (opening) Describe() searchsource.Description { return getModel() }

func (o opening) Embed(ctx context.Context, _ []string) ([][]float32, error) {
	o.asked <- struct{}{}
	<-ctx.Done()
	return nil, ctx.Err()
}

func (opening) Close() error { return nil }

func TestASearchByWordsAnswersWhileTheSearchByMeaningWaitsForItsSource(t *testing.T) {
	c := newCorpus(t)
	var errs []error
	model := opening{asked: make(chan struct{}, 1)}
	finds := search.New(c.db.ChunkSearch(), vaultreader.Readers{}, nil, c.participants(model), func(err error) { errs = append(errs, err) })

	meaningCtx, cancelMeaning := context.WithCancel(t.Context())
	meaning := make(chan error, 1)
	go func() {
		_, err := finds.Execute(meaningCtx, c.first, "disorder", search.GetTypingParameters(search.Sources, 8))
		meaning <- err
	}()
	<-model.asked

	found, err := finds.Execute(t.Context(), c.first, "disorder", search.GetTypingParameters(search.Lexical, 8))
	if err != nil || len(found) != 1 || !strings.Contains(found[0].Source, "Entropy") {
		t.Fatalf("the words found %v, %v while the meaning waited", sources(found), err)
	}

	cancelMeaning()
	if err := <-meaning; !errors.Is(err, context.Canceled) {
		t.Errorf("the meaning search a newer query stopped returned %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("the stopped meaning search was reported as a failure of the source: %v", errs)
	}
}
