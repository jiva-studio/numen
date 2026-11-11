package search

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/libs/note/domain"

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

// Index is what a search asks of the index: `Passages` answers by words and by section names,
// `Nearest` answers by meaning.
type Index struct {
	Passages sourceport.PassageQueries
	Nearest  sourceport.NearestQueries
}

// VaultFinding resolves what the user typed, a name, a path or an identity, into a vault on the list.
type VaultFinding = func(nameOrPath string) (domain.Vault, error)

// VaultPreparing prepares a Search for a vault once it has been found.
type VaultPreparing = func(ctx context.Context, v domain.Vault) (Search, error)

// Search answers a query within one vault. Without an embedder only the word search runs.
// Floor gives the floor of a query that names none, under the recipe of the model that embeds it.
// Extractors is how many sources are extracted at once; the default holds where it is not positive.
type Search struct {
	find         VaultFinding
	prepare      VaultPreparing
	passages     sourceport.PassageQueries
	nearest      FindNearest
	readers      vaultport.VaultReaders
	embedder     embeddingport.Embedder
	queries      QueryEmbedder
	texts        textdomain.Readers
	Floor        FloorSource
	errorHandler window.ErrorHandler
	cache        queryCaching
	Extractors   int
}

// NewByName creates a Search that resolves the vault and prepares the search over it.
func NewByName(find VaultFinding, prepare VaultPreparing) Search {
	return Search{find: find, prepare: prepare}
}

// ExecuteByName resolves the vault and runs the search.
func (u Search) ExecuteByName(ctx context.Context, nameOrPath, query string, p Parameters) ([]domain.Passage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v, err := u.find(nameOrPath)
	if err != nil {
		return nil, err
	}
	runner := u
	if u.prepare != nil {
		var prepErr error
		runner, prepErr = u.prepare(ctx, v)
		if prepErr != nil {
			return nil, prepErr
		}
	}
	return runner.Execute(ctx, v, query, p)
}

// New returns a search over one vault's index.
// A nil embedder means the installation has no model: the meaning search does
// not run and the word search answers alone, since the vector index is optional.
// A nil `texts` means only notes return their text. `queries` embeds the queries.
// `errorHandler` receives the error of a search mode that could not answer; nil ignores such errors.
func New(index Index, readers vaultport.VaultReaders, texts textdomain.Readers, embedder embeddingport.Embedder, queries QueryEmbedder, errorHandler window.ErrorHandler) Search {
	if errorHandler == nil {
		errorHandler = func(error) {}
	}
	return Search{
		passages:     index.Passages,
		nearest:      NewFindNearest(index.Nearest),
		readers:      readers,
		embedder:     embedder,
		queries:      queries,
		texts:        texts,
		errorHandler: errorHandler,
		cache:        NewQueryCache(defaultCacheCapacity),
		Extractors:   defaultExtractors,
	}
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
	return u.nearest.Execute(ctx, v.ID, sourcedomain.NearestQuery{Recipe: recipe, Vector: vector, Kinds: p.Kinds, PathPrefix: p.PathPrefix, Limit: p.Dense, Floor: u.getFloor(ctx, v, recipe, p)})
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

func (u Search) getFloor(ctx context.Context, v domain.Vault, recipe string, p Parameters) float64 {
	switch {
	case p.Floor > 0:
		return p.Floor
	case u.Floor != nil:
		return u.Floor(ctx, v, recipe)
	default:
		return DefaultFloor
	}
}

// errUnreadable marks a source that exists but yields no usable text. It is not
// a search failure: the passage is dropped and the others are returned.
var errUnreadable = errors.New("nothing could be read from the source")

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

// queryCaching keeps the vectors of the queries embedded before.
type queryCaching interface {
	Get(recipe, query string) ([]float32, bool)
	Put(recipe, query string, vector []float32)
}
