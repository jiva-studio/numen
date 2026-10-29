package search

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
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

// DefaultFloor is the minimum cosine similarity at which a passage counts as
// an answer.
//
// The value is specific to the embedding model that produced the vectors: it is
// the similarity that model assigns to texts about different subjects.
const DefaultFloor = 0.50

// Parameters selects the search modes, the number of candidates each mode keeps
// and the minimum score a candidate must reach. A mode with no candidates does
// not run; when no mode is configured, all modes run.
type Parameters struct {
	// Limit is how many results come back.
	Limit int
	// Each is the maximum number of passages returned per document. Zero means
	// one. A result list shows one line per book.
	Each int
	// Lexical is how many candidates the words keep, and Dense how many passages
	// the meaning returns. The coarse pass under it keeps several
	// times as many, and the full-precision vectors order those.
	Lexical int
	Dense   int
	// Named is how many sections the name search keeps. A document has far fewer
	// sections than chunks and a name match is a strong signal, so a small number
	// suffices.
	Named int
	// Floor is the minimum similarity for a passage to count as an answer. Only
	// the meaning search reads it; the word search has no distance.
	Floor float64
	// IsGrowing marks the last word of the query as possibly incomplete, so the
	// index matches it as a prefix. A complete query is matched exactly.
	IsGrowing bool
	// Kinds are the source kinds the query targets. Empty means every kind.
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
// New sets every field. Without an embedder only the word search runs, and
// vector-based results are absent.
type Search struct {
	passages     sourceport.PassageQueries
	readers      vaultport.VaultReaders
	embedder     embeddingport.Embedder
	texts        textport.TextReaders
	floor        float64
	errorHandler windowport.ErrorHandler
	cache        *QueryCache
	// extractors is how many sources are extracted at once. Zero is
	// defaultExtractors.
	extractors int
}

// New returns a search over one vault's index.
//
// The embedder converts a query into a vector. Nil means the installation has
// no model: the meaning search does not run and the word search answers alone,
// which is a complete search, since the vector index is optional and may never
// finish filling.
//
// `floor` is the minimum similarity for a passage to count as an answer, in the
// units of the model in use. Zero selects DefaultFloor.
//
// `texts` is the text versions that sources are cut from. Nil means the
// installation has none, and only notes return their text.
//
// `errorHandler` receives the error of a search mode that could not answer.
// Nil ignores such errors.
func New(passages sourceport.PassageQueries, readers vaultport.VaultReaders, texts textport.TextReaders, embedder embeddingport.Embedder, floor float64, errorHandler windowport.ErrorHandler) Search {
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
		texts:        texts,
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
	return u.readForQuery(ctx, v, query, collapse(merge(a.getRankings(p)...), a.named, p.Each, p.Limit))
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

// searchDense stores the passages nearest the meaning of the query. When the
// model is unreachable (no network, expired key) the word search answers alone.
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

// errNoVector reports that the query produced no vector, so the meaning search
// has nothing to search with.
var errNoVector = errors.New("the query was not turned into a vector")

// findNearest searches by meaning, using the vector of the query.
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
	// A vector is stored under the recipe that produced it, which covers every
	// property of the model that determines what a vector is. A lookup under any
	// other recipe finds no vector and this mode returns nothing.
	return u.passages.FindNearest(ctx, v.ID, recipe, vector, p.Kinds, p.Dense, p.Floor)
}

// read fills in the text of each passage from the vault. A chunk is a position
// in a file; the file holds the text.
//
// A passage whose file is no longer in the vault is dropped: the file is
// authoritative and the index follows it.
//
// Passages are grouped by source before reading, because a ranking scatters the
// passages of one source. Up to defaultExtractors sources are extracted
// concurrently, and each source's decoded text is released once its passages
// are filled in. The first error, or cancellation of ctx, stops the sources not
// yet started.
func (u Search) read(ctx context.Context, v domain.Vault, found []domain.Passage) ([]domain.Passage, error) {
	return u.readForQuery(ctx, v, "", found)
}

func (u Search) readForQuery(ctx context.Context, v domain.Vault, query string, found []domain.Passage) ([]domain.Passage, error) {
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
	of := sourceTexts{files: reader}
	if u.texts != nil {
		if of.texts, err = u.texts.Open(v); err != nil {
			return nil, err
		}
	}

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
			return fillSource(gctx, of, source, query, indices, found, filled, keep)
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
	of sourceTexts,
	source string,
	query string,
	indices []int,
	found, filled []domain.Passage,
	keep []bool,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	first := found[indices[0]]
	prose, version, err := of.getText(ctx, source, first.Producer)
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
		if version != nil {
			hitAt := p.HitAt
			if hitAt == 0 {
				hitAt = findHitOffset(p.Text, query)
			}
			p.Selector = version.GetSelector(p.Start + hitAt)
		}
		filled[i] = p
		keep[i] = true
	}
	return nil
}

// findHitOffset is the byte offset of the first query word in text, and zero where none appears.
func findHitOffset(text, query string) int {
	lower := strings.ToLower(text)
	best := len(text) + 1
	for _, word := range strings.Fields(strings.ToLower(query)) {
		if pos := strings.Index(lower, word); pos >= 0 && pos < best {
			best = pos
		}
	}
	if best > len(text) {
		return 0
	}
	return best
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

// errUnreadable marks a source that exists but yields no usable text. It is not
// a search failure: the passage is dropped and the others are returned.
var errUnreadable = errors.New("nothing could be read from the source")

// sourceTexts is where a passage's text is read from: a note's own file, and
// the text version every other source was cut from.
type sourceTexts struct {
	files vaultport.VaultReader
	texts textport.Texts
}

// getText returns the text a source's chunks address and the version it comes
// from, if any. A source without a version is a note, and its text is its file.
// A source with a version reads that version; a missing version is
// errUnreadable.
func (s sourceTexts) getText(ctx context.Context, path, version string) (string, *domain.VersionText, error) {
	if version == "" {
		raw, err := s.files.Read(ctx, path)
		if err != nil {
			return "", nil, err
		}
		return string(raw), nil, nil
	}
	if s.texts == nil {
		return "", nil, errUnreadable
	}
	doc, err := s.texts.GetVersionText(ctx, path, domain.TextVersionID(version))
	if errors.Is(err, textport.ErrNoText) {
		return "", nil, errUnreadable
	}
	if err != nil {
		return "", nil, err
	}
	return doc.Text, &doc, nil
}

// span is the text a chunk addresses, bounded by the current length of the
// source. A source edited after indexing is shorter than the chunk offsets
// indicate.
//
// The result is copied from the source's decoded text and not sliced from it,
// so a retained passage holds only the span and not the whole document.
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

// proseOpens is the byte offset at which a source's prose begins. A note's
// frontmatter precedes its prose; a text without frontmatter starts at byte 0.
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

// Mode selects one ranking of a search. A hybrid search fuses the rankings of
// all modes into one.
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

// GetTypingParameters returns the parameters for a search in the given mode
// while the query is being typed: the last word is matched as a prefix.
//
// Every mode other than the given one gets no candidates and does not run.
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
