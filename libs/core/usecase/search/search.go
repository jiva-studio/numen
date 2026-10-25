package search

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/libs/core/internal/text"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// defaultLimit is how many results a caller that names no number gets.
const defaultLimit = 20

// defaultExtractors is how many sources a search extracts at once when no limit
// is set. Each one holds its decoded text until its passages are cut, so it is
// also the most decoded sources a search holds.
const defaultExtractors = 4

// lexicalCandidates is how many candidates the words keep for every result
// the search returns, so the answer is among what it ranks.
const lexicalCandidates = 5

// DefaultFloor is how near the query a passage stands to be an answer, in
// cosine similarity.
//
// The number belongs to the model the vectors were made by: it is where that
// model puts two pieces of text about different things, and another model puts
// them somewhere else.
const DefaultFloor = 0.50

// Parameters says which ways a search is asked, how many candidates each keeps,
// and what a candidate has to reach. A half that keeps none does not run; with
// neither named, both run.
type Parameters struct {
	// Limit is how many results come back.
	Limit int
	// Each is how many passages one document may answer with. Zero is one, and
	// a document names itself once.
	//
	// about a thing, and a list a person runs their eye down needs one line per
	// book.
	Each int
	// Lexical is how many candidates the words keep, and Dense how many passages
	// the meaning returns. The coarse pass under it keeps several
	// times as many, and the full-precision vectors order those.
	Lexical int
	Dense   int
	// Named is how many sections the names keep. A document has far fewer
	// sections than chunks, and one name matching is a strong thing to have
	// said, so a few of them are enough.
	Named int
	// Floor is how near the query a passage stands to be an answer at all. It
	// is read only by the meaning half; the words half has no distance.
	Floor float64
	// IsGrowing says the last word typed may still be being typed, so the index
	// matches it by its opening. A question that is finished is asked exactly.
	IsGrowing bool
	// Kinds are the kinds of source the question is about. None is every kind,
	// which is what a question that says nothing about the sort of file it
	// wants asks for.
	//
	// A person asking a book about something is asking about the book.
	Kinds []domain.SourceKind
}

// fill supplies what the caller left out.
func (p Parameters) fill() Parameters {
	if p.Limit <= 0 {
		p.Limit = defaultLimit
	}
	if p.Each <= 0 {
		p.Each = 1
	}
	if p.Lexical <= 0 && p.Dense <= 0 && p.Named <= 0 {
		p.Lexical = p.Limit * lexicalCandidates
		p.Dense = p.Limit
		p.Named = p.Limit
	}
	if p.Floor == 0 {
		p.Floor = DefaultFloor
	}
	return p
}

// Search answers a query within one vault, over everything the vault holds.
//
// One database holds every vault, so the vault is an argument of the search.
// Every field is asked for by New. An embedder that is left out answers: the
// words half runs alone, with none of what the vectors hold.
type Search struct {
	passages     sourceport.PassageQueries
	readers      vaultport.VaultReaders
	embedder     embeddingport.Embedder
	derived      vaultport.DerivedStores
	documents    sourceport.TextExtractor
	floor        float64
	errorHandler windowport.ErrorHandler
	cache        *QueryCache
	// extractors is how many sources are extracted at once. Zero is
	// defaultExtractors.
	extractors int
}

// New is a search over one vault's index.
//
// The embedder turns a query into a vector. Nil is an installation with no
// model, and then the meaning half does not run and the words half answers alone,
// which is a whole search: the vector index is optional and may never finish
// filling.
//
// `floor` is how near the query a passage stands to be an answer, in the units
// the model in use measures in. Zero takes DefaultFloor.
//
// `errorHandler` hears about a half that could not answer. Nothing is said by
// passing nothing.
func New(passages sourceport.PassageQueries, readers vaultport.VaultReaders, derived vaultport.DerivedStores, documents sourceport.TextExtractor, embedder embeddingport.Embedder, floor float64, errorHandler windowport.ErrorHandler) Search {
	if floor == 0 {
		floor = DefaultFloor
	}
	if errorHandler == nil {
		errorHandler = func(error) {}
	}
	return Search{
		passages:     passages,
		readers:      readers,
		embedder:     embedder,
		derived:      derived,
		documents:    documents,
		floor:        floor,
		errorHandler: errorHandler,
		cache:        NewQueryCache(defaultCacheCapacity),
	}
}

// WithExtractors returns a search that extracts up to n sources at once. Zero or
// less is defaultExtractors.
func (u Search) WithExtractors(n int) Search {
	u.extractors = n
	return u
}

// WithCache returns a search that uses the query cache given.
func (u Search) WithCache(cache *QueryCache) Search {
	u.cache = cache
	return u
}

// Execute runs the ways the parameters name, merges their rankings by rank,
// and returns one passage per document.
//
// Three orders: the words in a chunk, the meaning of a chunk, and the name of
// the section a chunk opens. A chunk that two of them place well outranks one
// that any of them placed first alone.
func (u Search) Execute(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.Passage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.Floor == 0 {
		p.Floor = u.floor
	}
	p = p.fill()

	a, err := u.searchAll(ctx, v, query, p)
	if err != nil {
		return nil, err
	}
	return u.read(ctx, v, collapse(merge(a.getRankings(p)...), a.named, p.Each, p.Limit))
}

// answers is what each half of a search returned.
type answers struct {
	lexical  []domain.Passage
	named    []domain.Passage
	dense    []domain.Passage
	hasDense bool
}

// getRankings is the orders of the halves that ran, in the order they merge.
func (a *answers) getRankings(p Parameters) [][]domain.Passage {
	var out [][]domain.Passage
	if p.Lexical > 0 {
		out = append(out, a.lexical)
	}
	if p.Named > 0 {
		out = append(out, a.named)
	}
	if a.hasDense {
		out = append(out, a.dense)
	}
	return out
}

// searchAll runs the halves the parameters name at once and waits for all of them.
func (u Search) searchAll(ctx context.Context, v domain.Vault, query string, p Parameters) (*answers, error) {
	a := &answers{}
	g, gctx := errgroup.WithContext(ctx)
	if p.Lexical > 0 {
		g.Go(func() error { return u.searchLexical(gctx, v, query, p, a) })
	}
	if p.Named > 0 {
		g.Go(func() error { return u.searchNamed(gctx, v, query, p, a) })
	}
	if p.Dense > 0 && u.embedder != nil {
		g.Go(func() error { return u.searchDense(gctx, v, query, p, a) })
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return a, nil
}

// searchLexical stores the passages the words of the query match.
func (u Search) searchLexical(ctx context.Context, v domain.Vault, query string, p Parameters, a *answers) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	found, err := u.passages.FindLexical(ctx, v.ID, query, p.Kinds, p.Lexical, p.IsGrowing)
	if err != nil {
		return err
	}
	a.lexical = found
	return nil
}

// searchNamed stores the passages whose section names the query matches.
func (u Search) searchNamed(ctx context.Context, v domain.Vault, query string, p Parameters, a *answers) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	found, err := u.passages.GetNamedPassages(ctx, v.ID, query, p.Kinds, p.Named, p.IsGrowing)
	if err != nil {
		return err
	}
	a.named = found
	return nil
}

// searchDense stores the passages nearest the meaning of the query. A model out of
// reach leaves the words to answer, because a vault is searched on a machine
// with no network and by a person whose key has run out.
func (u Search) searchDense(ctx context.Context, v domain.Vault, query string, p Parameters, a *answers) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	found, err := u.findNearest(ctx, v, query, p)
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, errNoVector):
		u.errorHandler(err)
		return nil
	case err != nil:
		return err
	default:
		a.dense = found
		a.hasDense = true
		return nil
	}
}

// errNoVector marks the query having no vector, which is the half that asks by
// meaning having nothing to ask with.
var errNoVector = errors.New("the query was not turned into a vector")

// findNearest is the search asked by meaning, over a vector of the query itself.
func (u Search) findNearest(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.Passage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	recipe := u.embedder.GetModel().Recipe()
	var vector []float32
	if u.cache != nil {
		if cached, ok := u.cache.Get(recipe, query); ok {
			vector = cached
		}
	}
	if vector == nil {
		vectors, err := u.embedder.Embed(ctx, []string{query})
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errNoVector, err)
		}
		if len(vectors) != 1 {
			return nil, fmt.Errorf("%w: the embedder answered with %d vectors for one query", errNoVector, len(vectors))
		}
		vector = vectors[0]
		if u.cache != nil {
			u.cache.Put(recipe, query, vector)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// A vector is kept under the recipe it was made by, which is everything
	// about the model that decides what a vector is. Asked under anything else,
	// no vector is found and this half answers nothing at all.
	return u.passages.FindNearest(ctx, v.ID, recipe, vector, p.Kinds, p.Dense, p.Floor)
}

// read fills in the text of each passage from the vault. A chunk is a place in a
// file, and the file is what holds the words.
//
// A passage naming a file the vault no longer holds is dropped: the file is what
// is true, and the index follows it.
//
// A ranking scatters one source's passages wherever each chunk placed, so they
// are grouped by source before this reads any of them. Up to defaultExtractors
// sources are extracted at once, and each one's decoded text is released as soon
// as its own passages are filled in. The first error, or the end of ctx, stops
// the sources not yet started.
func (u Search) read(ctx context.Context, v domain.Vault, found []domain.Passage) ([]domain.Passage, error) {
	if len(found) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return nil, err
	}
	var store vaultport.DerivedStore
	if u.derived != nil {
		if store, err = u.derived.Open(v); err != nil {
			return nil, err
		}
	}
	of := text.Reader{Vault: reader, Derived: store, Documents: u.documents}

	groups, order := groupBySource(found)

	filled := make([]domain.Passage, len(found))
	keep := make([]bool, len(found))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(u.GetExtractors())
	for _, source := range order {
		if gctx.Err() != nil {
			break
		}
		indices := groups[source]
		g.Go(func() error {
			return fillSource(gctx, of, source, indices, found, filled, keep)
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	out := make([]domain.Passage, 0, len(found))
	for i, k := range keep {
		if k {
			out = append(out, filled[i])
		}
	}
	return out, nil
}

// GetExtractors is how many sources a search extracts at once.
func (u Search) GetExtractors() int {
	if u.extractors > 0 {
		return u.extractors
	}
	return defaultExtractors
}

// fillSource extracts one source and fills in the passages of `found` at
// `indices`, which name it. It writes only those indices of `filled` and `keep`.
func fillSource(
	ctx context.Context,
	of text.Reader,
	source string,
	indices []int,
	found, filled []domain.Passage,
	keep []bool,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	first := found[indices[0]]
	prose, err := extractText(ctx, of, source, first.Producer, first.SourceHash)
	if vaultport.IsNoNote(err) || errors.Is(err, errUnreadable) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", source, err)
	}
	opens := proseOpens(prose)
	for _, i := range indices {
		if err := ctx.Err(); err != nil {
			return err
		}
		p := found[i]
		p.Text = span(prose, p.Start, p.Length)
		p.Line = lineOf(prose, opens, p.Start+p.HitAt)
		filled[i] = p
		keep[i] = true
	}
	return nil
}

// groupBySource is the index of each passage in `found`, grouped by source and
// ordered by a source's first appearance.
func groupBySource(found []domain.Passage) (map[string][]int, []string) {
	groups := make(map[string][]int, len(found))
	order := make([]string, 0, len(found))
	for i, p := range found {
		if _, target := groups[p.Source]; !target {
			order = append(order, p.Source)
		}
		groups[p.Source] = append(groups[p.Source], i)
	}
	return groups, order
}

// errUnreadable is a source that is there and says nothing this can use. It is
// not a failure of the search: the passage is dropped and the rest answer.
var errUnreadable = errors.New("nothing could be read from the source")

// extractText is the text a source's chunks are places in.
//
// A note's text is its file. A book's is what taking the text out of it produces,
// and a chunk's offsets belong to that, not to the bytes on disk: slicing the
// archive at a text offset returns compressed noise. Extraction is deterministic,
// which is what lets the text be recovered from the source it belongs to.
//
// Which reader produces it is decided in one place, so that what a search slices
// and what an extractor cut are the same text.
func extractText(ctx context.Context, reader text.Reader, path, from, hash string) (string, error) {
	doc, err := reader.GetDocument(ctx, path, from, hash)
	if errors.Is(err, text.ErrUnreadable) {
		return "", errUnreadable
	}
	if err != nil {
		return "", err
	}
	return doc.Text, nil
}

// span is the text a chunk addresses, bounded by what the source holds now. A
// source edited since it was indexed is shorter than the chunk says.
//
// The result is copied out of the source's decoded text and not sliced from
// it, so a passage kept after this returns holds the span alone, not the whole
// document the span was cut from.
func span(raw string, start, length int) string {
	if start < 0 || start >= len(raw) || length <= 0 {
		return ""
	}
	end := start + length
	if end > len(raw) {
		end = len(raw)
	}
	return strings.Clone(raw[start:end])
}

// lineOf is the line a passage begins on, counted from the first line of the
// prose, which opens at `opens`.
func lineOf(raw string, opens, start int) int {
	if start > len(raw) {
		start = len(raw)
	}
	if start <= opens {
		return 0
	}
	return strings.Count(raw[opens:start], "\n")
}

// proseOpens is the byte a source's prose begins at. A note's frontmatter
// stands before its prose; a text that opens with none is prose from its first
// byte.
func proseOpens(raw string) int {
	rest := strings.TrimPrefix(raw, "\ufeff")
	first, after, found := strings.Cut(rest, "\n")
	if !found || strings.TrimRight(first, "\r") != "---" {
		return 0
	}
	for pos := 0; pos < len(after); {
		lineEnd := strings.IndexByte(after[pos:], '\n')
		var line string
		var next int
		if lineEnd >= 0 {
			line = after[pos : pos+lineEnd]
			next = pos + lineEnd + 1
		} else {
			line = after[pos:]
			next = len(after)
		}
		if strings.TrimRight(line, "\r") == "---" {
			return (len(raw) - len(after)) + next
		}
		pos = next
	}
	return 0
}

// Mode is how a search is asked. Each mode is an order of its own, and a
// hybrid search fuses them into one.
type Mode int

const (
	// Hybrid: all of them, fused into one ranking.
	Hybrid Mode = iota
	// Lexical: what is written, matched as words.
	Lexical
	// Dense: what the query means, against the vectors the index holds.
	Dense
	// ByName: the names of the sections a source divides into.
	ByName
)

// GetTypingParameters is the parameters for a search asked in the mode named,
// while a person is still typing it: the last word is matched by its opening.
//
// A mode that is not wanted keeps no candidates, which is how one is told not
// to run. Every mode but the one named is silenced, so a caller drawing them
// apart is shown one of them and not one and a half.
func GetTypingParameters(mode Mode, limit int) Parameters {
	p := Parameters{Limit: limit, IsGrowing: true}.fill()
	switch mode {
	case Lexical:
		p.Dense, p.Named = 0, 0
	case Dense:
		p.Lexical, p.Named = 0, 0
	case ByName:
		p.Lexical, p.Dense = 0, 0
	case Hybrid:
	}
	return p
}
