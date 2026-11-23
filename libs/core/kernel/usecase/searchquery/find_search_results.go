package searchquery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ErrNegativeLimit is a request for a negative count of results.
var ErrNegativeLimit = errors.New("the limit of results is negative")

// FindSearchResultsDeps is what FindSearchResults reads. A zero Limits takes the defaults, and the zone of
// Clock reads a date written without an offset. Catalog returns the catalog a query is parsed against, the built-in
// one when nil. Evaluate returns the results of a parsed query, FillDocuments
// sets the title, the time of change and the size of the document of each result, and Stages applies the
// stages of a query to its results.
type FindSearchResultsDeps struct {
	Parser        QueryParser
	GetCatalog    func() searchquery.Catalog
	Clock         machineport.Clock
	Evaluate      func(ctx context.Context, v domain.Vault, q searchquery.Query, zone *time.Location, limits searchquery.Limits, generation string) (searchquery.SearchResults, error)
	FillDocuments func(ctx context.Context, v domain.Vault, results []searchquery.SearchResult) error
	RunStages     func(ctx context.Context, v domain.Vault, q searchquery.Query, r searchquery.SearchResults, req searchquery.Request) (searchquery.SearchResults, error)
	Limits        searchquery.Limits
}

// FindSearchResults parses a query and evaluates it over the index of one vault. Limits bounds every query.
type FindSearchResults struct {
	parser        QueryParser
	getCatalog    func() searchquery.Catalog
	clock         machineport.Clock
	evaluate      func(ctx context.Context, v domain.Vault, q searchquery.Query, zone *time.Location, limits searchquery.Limits, generation string) (searchquery.SearchResults, error)
	fillDocuments func(ctx context.Context, v domain.Vault, results []searchquery.SearchResult) error
	applyStages   func(ctx context.Context, v domain.Vault, q searchquery.Query, r searchquery.SearchResults, req searchquery.Request) (searchquery.SearchResults, error)
	Limits        searchquery.Limits
}

// NewFindSearchResults creates the use case.
func NewFindSearchResults(deps FindSearchResultsDeps) *FindSearchResults {
	limits := deps.Limits
	if limits == (searchquery.Limits{}) {
		limits = searchquery.DefaultLimits()
	}
	catalog := deps.GetCatalog
	if catalog == nil {
		catalog = searchquery.GetBuiltinCatalog
	}
	return &FindSearchResults{
		parser: deps.Parser, getCatalog: catalog, clock: deps.Clock, evaluate: deps.Evaluate, fillDocuments: deps.FillDocuments, applyStages: deps.RunStages,
		Limits: limits,
	}
}

// Execute returns the results of the query. A query that is not valid returns *searchquery.ParseError, an
// FTS5 expression the index rejects included; a limit exceeded searchquery.ErrLimitExceeded; meaning with no
// model searchquery.ErrNoModel; a rerank stage with no row searchquery.ErrNoReranker and with every row
// failing *searchquery.RerankError; cancellation ctx.Err().
func (u *FindSearchResults) Execute(ctx context.Context, v domain.Vault, r searchquery.Request) (searchquery.SearchResults, error) {
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
	results, err := u.evaluate(ctx, v, q, zone, u.Limits, r.IndexGeneration)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	if results, err = u.runStages(ctx, v, q, results, r); err != nil {
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
	ctx context.Context, v domain.Vault, q searchquery.Query, results searchquery.SearchResults, r searchquery.Request,
) (searchquery.SearchResults, error) {
	if len(q.Stages) == 0 {
		return results, nil
	}
	if err := u.fillDocuments(ctx, v, results.Results); err != nil {
		return searchquery.SearchResults{}, err
	}
	return u.applyStages(ctx, v, q, results, r)
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
	raw, err := u.parser.ParseQuery(text, u.getCatalog())
	if err != nil {
		return searchquery.Query{}, err
	}
	return searchquery.Normalize(raw, u.Limits, zone)
}
