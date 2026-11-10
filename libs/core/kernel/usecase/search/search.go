package search

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/libs/note/domain"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
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

// DefaultFloor is the minimum cosine similarity at which a passage counts as an
// answer where neither the query, a stored floor nor the model names one.
// The similarity a model assigns to texts about different subjects is specific
// to the model, so the value is a starting point.
const DefaultFloor = 0.30

// FloorSource returns the floor for a vault under an embedding recipe.
type FloorSource func(ctx context.Context, v domain.Vault, recipe string) float64

// Parameters selects the search modes, the number of candidates each mode keeps
// and the minimum score a candidate must reach. A mode with no candidates does
// not run; when no mode is configured, all modes run.
type Parameters struct {
	Limit      int
	Each       int
	Lexical    int
	Dense      int
	Named      int
	Floor      float64
	IsGrowing  bool
	Kinds      []domain.SourceKind
	PathPrefix string
}

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
	floorSource  FloorSource
	errorHandler window.ErrorHandler
	cache        *QueryCache
	extractors   int
}

// New returns a search over one vault's index.
// A nil embedder means the installation has no model: the meaning search does
// not run and the word search answers alone, since the vector index is optional.
// A nil `texts` means only notes return their text. `errorHandler` receives the
// error of a search mode that could not answer; nil ignores such errors.
func New(passages sourceport.PassageQueries, readers vaultport.VaultReaders, texts textport.TextReaders, embedder embeddingport.Embedder, errorHandler window.ErrorHandler) Search {
	if errorHandler == nil {
		errorHandler = func(error) {}
	}
	return Search{
		passages:     passages,
		readers:      readers,
		embedder:     embedder,
		texts:        texts,
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

// WithFloorSource returns a search whose floor for a query is the source's, under
// the recipe of the model that embeds the query. It is consulted when the query
// names no floor.
func (u Search) WithFloorSource(source FloorSource) Search {
	u.floorSource = source
	return u
}

// WithCache returns a search that uses the query cache given.
func (u Search) WithCache(cache *QueryCache) Search {
	u.cache = cache
	return u
}

// Execute runs the ways the parameters name, merges their rankings by rank,
// and returns one passage per document.
// The orders are the words in a chunk, the meaning of a chunk and the name of
// the section a chunk opens. A chunk that two of them place well outranks one
// that any of them placed first alone.
func (u Search) Execute(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.Passage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
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

func (u Search) searchLexical(ctx context.Context, v domain.Vault, query string, p Parameters, a *answers) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	found, err := u.passages.FindLexical(ctx, v.ID, sourceport.LexicalQuery{Text: query, Kinds: p.Kinds, Limit: p.Lexical, IsGrowing: p.IsGrowing, PathPrefix: p.PathPrefix})
	if err != nil {
		return err
	}
	a.lexical = found
	return nil
}

func (u Search) searchNamed(ctx context.Context, v domain.Vault, query string, p Parameters, a *answers) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	found, err := u.passages.GetNamedPassages(ctx, v.ID, sourceport.LexicalQuery{Text: query, Kinds: p.Kinds, Limit: p.Named, IsGrowing: p.IsGrowing, PathPrefix: p.PathPrefix})
	if err != nil {
		return err
	}
	a.named = found
	return nil
}

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

func (u Search) findNearest(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.Passage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	recipe := u.embedder.GetModel().Recipe()
	vector, err := u.getQueryVector(ctx, recipe, query)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// A vector is stored under the recipe that produced it, which covers every
	// property of the model that determines what a vector is. A lookup under any
	// other recipe finds no vector and this mode returns nothing.
	return u.passages.FindNearest(ctx, v.ID, sourceport.NearestQuery{Recipe: recipe, Vector: vector, Kinds: p.Kinds, PathPrefix: p.PathPrefix, Limit: p.Dense, Floor: u.getFloor(ctx, v, recipe, p)})
}

func (u Search) getQueryVector(ctx context.Context, recipe, query string) ([]float32, error) {
	if u.cache != nil {
		if cached, ok := u.cache.Get(recipe, query); ok {
			return cached, nil
		}
	}
	vectors, err := embeddingport.EmbedQueries(ctx, u.embedder, []string{query})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errNoVector, err)
	}
	if len(vectors) != 1 {
		return nil, fmt.Errorf("%w: the embedder answered with %d vectors for one query", errNoVector, len(vectors))
	}
	if u.cache != nil {
		u.cache.Put(recipe, query, vectors[0])
	}
	return vectors[0], nil
}

func (u Search) getFloor(ctx context.Context, v domain.Vault, recipe string, p Parameters) float64 {
	switch {
	case p.Floor > 0:
		return p.Floor
	case u.floorSource != nil:
		return u.floorSource(ctx, v, recipe)
	default:
		return DefaultFloor
	}
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

	fill := passageFill{
		query: query, found: found,
		filled: make([]domain.Passage, len(found)), keep: make([]bool, len(found)),
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(u.GetExtractors())
	for _, source := range order {
		if gctx.Err() != nil {
			break
		}
		indices := groups[source]
		g.Go(func() error {
			return fillSource(gctx, of, source, indices, fill)
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	out := make([]domain.Passage, 0, len(found))
	for i, k := range fill.keep {
		if k {
			out = append(out, fill.filled[i])
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

// passageFill is what a search fills in: the passages found, the filled copies, the
// flags of those to keep, and the query that located them.
type passageFill struct {
	query         string
	found, filled []domain.Passage
	keep          []bool
}

func fillSource(
	ctx context.Context, of sourceTexts, source string, indices []int, fill passageFill,
) error {
	found, filled, keep, query := fill.found, fill.filled, fill.keep, fill.query
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

func findHitOffset(text, query string) int {
	lower := strings.ToLower(text)
	best := len(text) + 1
	for word := range strings.FieldsSeq(strings.ToLower(query)) {
		if pos := strings.Index(lower, word); pos >= 0 && pos < best {
			best = pos
		}
	}
	if best > len(text) {
		return 0
	}
	return best
}

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
	if errors.Is(err, textdomain.ErrNoText) {
		return "", nil, errUnreadable
	}
	if err != nil {
		return "", nil, err
	}
	return doc.Text, &doc, nil
}

func span(raw string, start, length int) string {
	if start < 0 || start >= len(raw) || length <= 0 {
		return ""
	}
	end := min(start+length, len(raw))
	return strings.Clone(raw[start:end])
}

func lineOf(raw string, opens, start int) int {
	if start > len(raw) {
		start = len(raw)
	}
	if start <= opens {
		return 0
	}
	return strings.Count(raw[opens:start], "\n")
}

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
