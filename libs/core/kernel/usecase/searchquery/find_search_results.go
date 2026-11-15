package searchquery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/operationport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrNegativeLimit is a request for a negative count of results.
var ErrNegativeLimit = errors.New("the limit of results is negative")

// FindSearchResultsDeps is what FindSearchResults reads. A nil Source is an installation with no model, and
// a nil Texts reads the text of notes only. Calls, Codec, Rows, Resolve, Results and Clock serve the rerank
// stage, and the zone of Clock reads a date written without an offset. A zero Limits or StageLimits takes the
// defaults.
type FindSearchResultsDeps struct {
	Parser       QueryParser
	Documents    DocumentQueries
	Vectors      VectorQueries
	Places       sourceport.ChunkPlaces
	Matcher      sourceport.WordMatching
	Readers      vaultport.VaultReaders
	Texts        textdomain.Readers
	Source       searchsourceport.QuerySource
	Floors       FloorReader
	ContentTypes contenttypeport.ContentTypeTable
	Contents     NoteContents
	Calls        connectionport.ModelCalls
	Codec        RerankCodec
	Rows         RerankRows
	Resolve      connectionport.ResolveConnection
	Results      operationport.OperationResults
	Links        LinkReaders
	Clock        machineport.Clock
	Limits       searchquery.Limits
	StageLimits  searchquery.StageLimits
}

// FindSearchResults parses a query and evaluates it over the index of one vault. Limits bounds every query.
type FindSearchResults struct {
	parser       QueryParser
	index        indexReads
	clock        machineport.Clock
	links        LinkReaders
	matcher      sourceport.WordMatching
	readers      vaultport.VaultReaders
	texts        textdomain.Readers
	source       searchsourceport.QuerySource
	floors       FloorReader
	contentTypes contenttypeport.ContentTypeTable
	reranker     reranker
	Limits       searchquery.Limits
}

// NewFindSearchResults creates the use case over the index, .
func NewFindSearchResults(deps FindSearchResultsDeps) *FindSearchResults {
	limits, stageLimits := deps.Limits, deps.StageLimits
	if limits == (searchquery.Limits{}) {
		limits = searchquery.DefaultLimits()
	}
	if stageLimits == (searchquery.StageLimits{}) {
		stageLimits = searchquery.DefaultStageLimits()
	}
	return &FindSearchResults{
		parser: deps.Parser, clock: deps.Clock,
		index: indexReads{
			documents: deps.Documents, vectors: deps.Vectors, words: deps.Documents, places: deps.Places, contents: deps.Contents,
		},
		matcher: deps.Matcher, readers: deps.Readers, texts: deps.Texts,
		source: deps.Source, floors: deps.Floors, contentTypes: deps.ContentTypes, links: deps.Links,
		reranker: reranker{
			calls: deps.Calls, codec: deps.Codec, rows: deps.Rows, resolve: deps.Resolve, results: deps.Results, clock: deps.Clock, limits: stageLimits,
		},
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
	if hasMeaning(q.Conditions) && !u.hasModel() {
		return searchquery.SearchResults{}, searchquery.ErrNoModel
	}
	req := u.newRequest(ctx, v, q, zone)
	warns := &warnings{}
	docs := newDocumentReader(u.index, u.links, u.contentTypes, req, warns)
	results, err := u.run(ctx, req, docs, warns)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	if results, err = u.runStages(ctx, docs, v, q, results, r); err != nil {
		return searchquery.SearchResults{}, err
	}
	results = searchquery.ApplyTop(results, u.getLimit(r.Limit, q.HasStage(searchquery.StageTop)))
	if len(q.Stages) == 0 {
		if err := docs.fillDocuments(ctx, results.Results); err != nil {
			return searchquery.SearchResults{}, err
		}
	}
	return results, ctx.Err()
}

func (u *FindSearchResults) runStages(
	ctx context.Context, docs *documentReader, v domain.Vault, q searchquery.Query, results searchquery.SearchResults,
	r searchquery.Request,
) (searchquery.SearchResults, error) {
	if len(q.Stages) == 0 {
		return results, nil
	}
	if err := docs.fillDocuments(ctx, results.Results); err != nil {
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

type indexReads struct {
	documents DocumentQueries
	vectors   VectorQueries
	words     WordQueries
	places    sourceport.ChunkPlaces
	contents  NoteContents
}
