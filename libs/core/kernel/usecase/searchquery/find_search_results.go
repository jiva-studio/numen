package searchquery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrNegativeLimit is a request for a negative count of results.
var ErrNegativeLimit = errors.New("the limit of results is negative")

// Request is one query. Limit is the count of results: zero takes Limits.DefaultLimit, a count above
// Limits.MaxLimit is cut to it.
type Request struct {
	Text  string
	Limit int
}

// FindSearchResultsDeps is what FindSearchResults reads. The zone of Clock reads a date written without an
// offset. Evaluate returns the results of a parsed query, and FillTitles sets the title of the document of each
// note result.
type FindSearchResultsDeps struct {
	Parser     QueryParser
	Clock      machineport.Clock
	Evaluate   func(ctx context.Context, v domain.Vault, q searchquery.Query, zone *time.Location, limits searchquery.Limits) (searchquery.SearchResults, error)
	FillTitles func(ctx context.Context, v domain.Vault, results []searchquery.SearchResult) error
}

// FindSearchResults parses a query and evaluates it over the index of one vault. Limits bounds every query.
type FindSearchResults struct {
	parser     QueryParser
	clock      machineport.Clock
	evaluate   func(ctx context.Context, v domain.Vault, q searchquery.Query, zone *time.Location, limits searchquery.Limits) (searchquery.SearchResults, error)
	fillTitles func(ctx context.Context, v domain.Vault, results []searchquery.SearchResult) error
	Limits     searchquery.Limits
}

// NewFindSearchResults creates the use case with the default limits.
func NewFindSearchResults(deps FindSearchResultsDeps) *FindSearchResults {
	return &FindSearchResults{
		parser: deps.Parser, clock: deps.Clock, evaluate: deps.Evaluate, fillTitles: deps.FillTitles,
		Limits: searchquery.DefaultLimits(),
	}
}

// Execute returns the results of the query. A query that is not valid returns *searchquery.ParseError, an
// FTS5 expression the index rejects included; a limit exceeded searchquery.ErrLimitExceeded; meaning with no
// model searchquery.ErrNoModel; cancellation ctx.Err().
func (u *FindSearchResults) Execute(ctx context.Context, v domain.Vault, r Request) (searchquery.SearchResults, error) {
	if err := ctx.Err(); err != nil {
		return searchquery.SearchResults{}, err
	}
	limit, err := u.getLimit(r.Limit)
	if err != nil {
		return searchquery.SearchResults{}, err
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
	if len(results.Results) > limit {
		results.Results = results.Results[:limit]
	}
	if err := u.fillTitles(ctx, v, results.Results); err != nil {
		return searchquery.SearchResults{}, err
	}
	return results, ctx.Err()
}

func (u *FindSearchResults) getLimit(limit int) (int, error) {
	switch {
	case limit < 0:
		return 0, fmt.Errorf("%w: %d", ErrNegativeLimit, limit)
	case limit == 0:
		return u.Limits.DefaultLimit, nil
	}
	return min(limit, u.Limits.MaxLimit), nil
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
