package searchquery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
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

// FindSearchResultsDeps is what FindSearchResults reads. A nil Source is an installation with no model, and
// a nil Texts reads the text of notes only. The zone of Clock reads a date written without an offset.
type FindSearchResultsDeps struct {
	Parser       QueryParser
	Chunks       ChunkIndex
	Matcher      sourceport.WordMatching
	Readers      vaultport.VaultReaders
	Texts        textdomain.Readers
	Source       searchsourceport.QuerySource
	Floors       FloorReader
	ContentTypes contenttypeport.ContentTypeTable
	Contents     NoteContents
	Links        LinkReaders
	Clock        machineport.Clock
}

// FindSearchResults parses a query and evaluates it over the index of one vault. Limits bounds every query.
type FindSearchResults struct {
	parser       QueryParser
	chunks       ChunkIndex
	contents     NoteContents
	clock        machineport.Clock
	links        LinkReaders
	matcher      sourceport.WordMatching
	readers      vaultport.VaultReaders
	texts        textdomain.Readers
	source       searchsourceport.QuerySource
	floors       FloorReader
	contentTypes contenttypeport.ContentTypeTable
	Limits       searchquery.Limits
}

// NewFindSearchResults creates the use case over the index, with the default limits.
func NewFindSearchResults(deps FindSearchResultsDeps) *FindSearchResults {
	return &FindSearchResults{
		parser: deps.Parser, clock: deps.Clock, chunks: deps.Chunks, contents: deps.Contents,
		matcher: deps.Matcher, readers: deps.Readers, texts: deps.Texts,
		source: deps.Source, floors: deps.Floors, contentTypes: deps.ContentTypes, links: deps.Links,
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
	if hasMeaning(q.Conditions) && !u.hasModel() {
		return searchquery.SearchResults{}, searchquery.ErrNoModel
	}
	req := u.newRequest(ctx, v, q, zone)
	warns := &warnings{}
	docs := newDocumentReader(u.chunks, u.contents, u.links, u.contentTypes, req, warns)
	results, err := u.run(ctx, req, docs, warns)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	if len(results.Results) > limit {
		results.Results = results.Results[:limit]
	}
	if err := docs.fillTitles(ctx, results.Results); err != nil {
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

func hasMeaning(conditions []searchquery.Condition) bool {
	for _, c := range conditions {
		if c.IsMeaning {
			return true
		}
	}
	return false
}

func (u *FindSearchResults) hasModel() bool {
	return u.source != nil && u.source.Describe().IsDescribed()
}
