package search

import (
	"context"
	"errors"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/libs/note/domain"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
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
// answer where neither the query, a stored floor nor the search source names one.
// The similarity a source assigns to texts about different subjects is specific
// to the source, so the value is a starting point.
const DefaultFloor = 0.30

// Parameters selects the participants of a search, the number of candidates each keeps
// and the floor the query names. A participant with no candidates does not run; when none is
// configured, all of them run. Lexical is the words of the index, Named the names of its sections,
// and Sources every search source a plugin provides.
type Parameters struct {
	Limit      int
	Each       int
	Lexical    int
	Sources    int
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
	if p.Lexical <= 0 && p.Sources <= 0 && p.Named <= 0 {
		p.Lexical = p.Limit * lexicalCandidates
		p.Sources = p.Limit
		p.Named = p.Limit
	}
	return p
}

// Query is what a search asks one participant: the text, the filters, how many passages to rank and
// the floor the query names, zero where the participant applies its own.
type Query struct {
	Text       string
	Kinds      []domain.SourceKind
	PathPrefix string
	Limit      int
	IsGrowing  bool
	Floor      float64
}

// Ranking is the answer of one participant: passages best first. In a leading ranking the first
// passage of a source is shown before the other passages of that source.
type Ranking struct {
	Passages  []domain.Passage
	IsLeading bool
}

// Participant answers a query with a ranking of passages. The words and the section names of the index
// are participants of the core; a search source of a plugin is another, and all are merged the same way.
type Participant interface {
	Rank(ctx context.Context, v domain.Vault, q Query) (Ranking, error)
}

// VaultFinding resolves what the user typed, a name, a path or an identity, into a vault on the list.
type VaultFinding = func(nameOrPath string) (domain.Vault, error)

// VaultPreparing prepares a Search for a vault once it has been found.
type VaultPreparing = func(ctx context.Context, v domain.Vault) (Search, error)

// Search answers a query within one vault. Without a search source only the participants of the core run.
// Extractors is how many sources are extracted at once; the default holds where it is not positive.
type Search struct {
	find         VaultFinding
	prepare      VaultPreparing
	words        Participant
	names        Participant
	sources      []Participant
	readers      vaultport.VaultReaders
	texts        textdomain.Readers
	errorHandler window.ErrorHandler
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

// New returns a search over one vault's index: its words and section names, and the search sources given.
// No source means the installation has none, and the participants of the core answer alone.
// A nil `texts` means only notes return their text.
// `errorHandler` receives the error of a search source that could not answer; nil ignores such errors.
func New(passages sourceport.PassageQueries, readers vaultport.VaultReaders, texts textdomain.Readers, sources []Participant, errorHandler window.ErrorHandler) Search {
	if errorHandler == nil {
		errorHandler = func(error) {}
	}
	return Search{
		words:        NewWords(passages),
		names:        NewNames(passages),
		sources:      sources,
		readers:      readers,
		texts:        texts,
		errorHandler: errorHandler,
		Extractors:   defaultExtractors,
	}
}

// Execute asks the participants the parameters name, merges their rankings by rank,
// and returns one passage per document.
// A chunk that two of them place well outranks one that any of them placed first alone. The rankings are
// fused by reciprocal rank fusion, which does not know which participant answered which ranking. The answer
// keeps at most p.Each passages of one source and p.Limit in all.
func (u Search) Execute(ctx context.Context, v domain.Vault, query string, p Parameters) ([]domain.Passage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p = p.fill()

	rankings, err := u.rankAll(ctx, v, query, p)
	if err != nil {
		return nil, err
	}
	return u.readForQuery(ctx, v, query, collapse(merge(rankings...), rankings, p.Each, p.Limit))
}

// asked is one participant a search asks, with the number of passages it ranks and whether its failure
// fails the search.
type asked struct {
	participant Participant
	limit       int
	isTolerated bool
}

func (u Search) getAskedParticipants(p Parameters) []asked {
	var out []asked
	if p.Lexical > 0 && u.words != nil {
		out = append(out, asked{participant: u.words, limit: p.Lexical})
	}
	if p.Named > 0 && u.names != nil {
		out = append(out, asked{participant: u.names, limit: p.Named})
	}
	if p.Sources > 0 {
		for _, source := range u.sources {
			out = append(out, asked{participant: source, limit: p.Sources, isTolerated: true})
		}
	}
	return out
}

func (u Search) rankAll(ctx context.Context, v domain.Vault, query string, p Parameters) ([]Ranking, error) {
	asks := u.getAskedParticipants(p)
	answers := make([]*Ranking, len(asks))
	g, gctx := errgroup.WithContext(ctx)
	for i, one := range asks {
		g.Go(func() error {
			ranking, err := u.rank(gctx, v, one, Query{
				Text: query, Kinds: p.Kinds, PathPrefix: p.PathPrefix, Limit: one.limit, IsGrowing: p.IsGrowing, Floor: p.Floor,
			})
			answers[i] = ranking
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	out := make([]Ranking, 0, len(answers))
	for _, one := range answers {
		if one != nil {
			out = append(out, *one)
		}
	}
	return out, nil
}

func (u Search) rank(ctx context.Context, v domain.Vault, one asked, q Query) (*Ranking, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ranking, err := one.participant.Rank(ctx, v, q)
	switch {
	case err == nil:
		return &ranking, nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return nil, err
	case one.isTolerated:
		u.errorHandler(err)
		return nil, nil
	default:
		return nil, err
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
	// Sources: the search sources plugins provide.
	Sources
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
		p.Sources, p.Named = 0, 0
	case Sources:
		p.Lexical, p.Named = 0, 0
	case ByName:
		p.Lexical, p.Sources = 0, 0
	case Hybrid:
	}
	return p
}
