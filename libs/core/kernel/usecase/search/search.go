package search

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
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

// QueryEmbedder returns one vector per query, made for searching with the query as the question.
type QueryEmbedder interface {
	Execute(ctx context.Context, queries []string) ([][]float32, error)
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
	queries      QueryEmbedder
	texts        textdomain.Readers
	floorSource  FloorSource
	errorHandler window.ErrorHandler
	cache        *QueryCache
	extractors   int
}

// New returns a search over one vault's index.
// A nil embedder means the installation has no model: the meaning search does
// not run and the word search answers alone, since the vector index is optional.
// A nil `texts` means only notes return their text. `queries` embeds the queries.
// `errorHandler` receives the error of a search mode that could not answer; nil ignores such errors.
func New(passages sourceport.PassageQueries, readers vaultport.VaultReaders, texts textdomain.Readers, embedder embeddingport.Embedder, queries QueryEmbedder, errorHandler window.ErrorHandler) Search {
	if errorHandler == nil {
		errorHandler = func(error) {}
	}
	return Search{
		passages:     passages,
		readers:      readers,
		embedder:     embedder,
		queries:      queries,
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
	found, err := u.passages.FindLexical(ctx, v.ID, sourcedomain.LexicalQuery{Text: query, Kinds: p.Kinds, Limit: p.Lexical, IsGrowing: p.IsGrowing, PathPrefix: p.PathPrefix})
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
	found, err := u.passages.GetNamedPassages(ctx, v.ID, sourcedomain.LexicalQuery{Text: query, Kinds: p.Kinds, Limit: p.Named, IsGrowing: p.IsGrowing, PathPrefix: p.PathPrefix})
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
	return u.passages.FindNearest(ctx, v.ID, sourcedomain.NearestQuery{Recipe: recipe, Vector: vector, Kinds: p.Kinds, PathPrefix: p.PathPrefix, Limit: p.Dense, Floor: u.getFloor(ctx, v, recipe, p)})
}

func (u Search) getQueryVector(ctx context.Context, recipe, query string) ([]float32, error) {
	if u.cache != nil {
		if cached, ok := u.cache.Get(recipe, query); ok {
			return cached, nil
		}
	}
	vectors, err := u.queries.Execute(ctx, []string{query})
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

