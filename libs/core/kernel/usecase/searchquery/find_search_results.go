package searchquery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrNegativeLimit is a request for a negative count of results.
var ErrNegativeLimit = errors.New("the limit of results is negative")

// Request is one query. Limit is the count of results: zero takes Limits.DefaultLimit, or no cut when the
// query has a top stage; a count above Limits.MaxLimit is cut to it. Priority is whose action the calls of
// a rerank stage serve.
type Request struct {
	Text     string
	Limit    int
	Priority modelcall.Priority
}

// FindSearchResultsDeps is what FindSearchResults reads. A zero Limits takes the defaults, and the zone of
// Clock reads a date written without an offset. Evaluate returns the results of a parsed query, FillDocuments
// sets the title, the time of change and the size of the document of each result, and Stages applies the
// stages of a query to its results.
type FindSearchResultsDeps struct {
	Parser        QueryParser
	Clock         machineport.Clock
	Evaluate      func(ctx context.Context, v domain.Vault, q searchquery.Query, zone *time.Location, limits searchquery.Limits) (searchquery.SearchResults, error)
	FillDocuments func(ctx context.Context, v domain.Vault, results []searchquery.SearchResult) error
	Stages        func(ctx context.Context, v domain.Vault, q searchquery.Query, r searchquery.SearchResults, priority modelcall.Priority) (searchquery.SearchResults, error)
	Limits        searchquery.Limits
}

// FindSearchResults parses a query and evaluates it over the index of one vault. Limits bounds every query.
type FindSearchResults struct {
	parser        QueryParser
	clock         machineport.Clock
	evaluate      func(ctx context.Context, v domain.Vault, q searchquery.Query, zone *time.Location, limits searchquery.Limits) (searchquery.SearchResults, error)
	fillDocuments func(ctx context.Context, v domain.Vault, results []searchquery.SearchResult) error
	stages        func(ctx context.Context, v domain.Vault, q searchquery.Query, r searchquery.SearchResults, priority modelcall.Priority) (searchquery.SearchResults, error)
	Limits        searchquery.Limits
}

// NewFindSearchResults creates the use case.
func NewFindSearchResults(deps FindSearchResultsDeps) *FindSearchResults {
	limits := deps.Limits
	if limits == (searchquery.Limits{}) {
		limits = searchquery.DefaultLimits()
	}
	return &FindSearchResults{
		parser: deps.Parser, clock: deps.Clock, evaluate: deps.Evaluate, fillDocuments: deps.FillDocuments, stages: deps.Stages,
		Limits: limits,
	}
}

// Execute returns the results of the query. A query that is not valid returns *searchquery.ParseError, an
// FTS5 expression the index rejects included; a limit exceeded searchquery.ErrLimitExceeded; meaning with no
// model searchquery.ErrNoModel; a rerank stage with no row searchquery.ErrNoReranker and with every row
// failing *searchquery.RerankError; cancellation ctx.Err().
func (u *FindSearchResults) Execute(ctx context.Context, v domain.Vault, r Request) (searchquery.SearchResults, error) {
	if err := ctx.Err(); err != nil {
		return searchquery.SearchResults{}, err
	}
	if r.Limit < 0 {
		return searchquery.SearchResults{}, fmt.Errorf("%w: %d", ErrNegativeLimit, r.Limit)
	}
	zone := u.clock.Now().Location()
	q, err := u.parse(r.Text, zone)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	results, err := u.evaluate(ctx, v, q, zone, u.Limits)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	if results, err = u.runStages(ctx, v, q, results, r.Priority); err != nil {
		return searchquery.SearchResults{}, err
	}
	results = searchquery.ApplyTop(results, u.getLimit(r.Limit, q.HasStage(searchquery.StageTop)))
	if len(q.Stages) == 0 {
		if err := u.fillDocuments(ctx, v, results.Results); err != nil {
			return searchquery.SearchResults{}, err
		}
	}
	return results, ctx.Err()
}

func (u *FindSearchResults) runStages(
	ctx context.Context, v domain.Vault, q searchquery.Query, results searchquery.SearchResults, priority modelcall.Priority,
) (searchquery.SearchResults, error) {
	if len(q.Stages) == 0 {
		return results, nil
	}
	if err := u.fillDocuments(ctx, v, results.Results); err != nil {
		return searchquery.SearchResults{}, err
	}
	return u.stages(ctx, v, q, results, priority)
}

func (u *FindSearchResults) getLimit(limit int, hasTop bool) int {
	switch {
	case limit == 0 && hasTop:
		return u.Limits.MaxLimit
	case limit == 0:
		return u.Limits.DefaultLimit
	}
	return min(limit, u.Limits.MaxLimit)
}

func (u *FindSearchResults) parse(text string, zone *time.Location) (searchquery.Query, error) {
	if err := searchquery.CheckQueryText(text, u.Limits); err != nil {
		return searchquery.Query{}, err
	}
	raw, err := u.parser.ParseQuery(text, searchquery.GetBuiltinCatalog())
	if err != nil {
		return searchquery.Query{}, err
	}
	return searchquery.Normalize(raw, u.Limits, zone)
}
