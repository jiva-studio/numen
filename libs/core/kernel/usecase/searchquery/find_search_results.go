package searchquery

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
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

// Request is one query. Limit is the count of results: zero takes Limits.DefaultLimit, or no cut when the
// query has a top stage; a count above Limits.MaxLimit is cut to it. Priority is whose action the calls of
// a rerank stage serve.
type Request struct {
	Text     string
	Limit    int
	Priority modelcall.Priority
}

// FindSearchResultsDeps is what FindSearchResults reads. A nil Source is an installation with no model, and
// a nil Texts reads the text of notes only. Calls, Codec, Rows, Resolve, Results and Clock serve the rerank
// stage. A zero Limits or StageLimits takes the defaults.
type FindSearchResultsDeps struct {
	Parser       QueryParser
	Documents    DocumentQueries
	Vectors      VectorQueries
	Words        WordQueries
	Places       sourceport.ChunkPlaces
	Matcher      sourceport.WordMatching
	Readers      vaultport.VaultReaders
	Texts        textdomain.Readers
	Source       searchsourceport.QuerySource
	Floors       FloorReader
	ContentTypes contenttypeport.ContentTypeTable
	Calls        connectionport.ModelCalls
	Codec        RerankCodec
	Rows         RerankRows
	Resolve      connectionport.ResolveConnection
	Results      operationport.OperationResults
	Clock        machineport.Clock
	Limits       searchquery.Limits
	StageLimits  searchquery.StageLimits
}

// FindSearchResults parses a query and evaluates it over the index of one vault. Limits bounds every query.
type FindSearchResults struct {
	parser       QueryParser
	documents    DocumentQueries
	vectors      VectorQueries
	words        WordQueries
	places       sourceport.ChunkPlaces
	matcher      sourceport.WordMatching
	readers      vaultport.VaultReaders
	texts        textdomain.Readers
	source       searchsourceport.QuerySource
	floors       FloorReader
	contentTypes contenttypeport.ContentTypeTable
	reranker     reranker
	Limits       searchquery.Limits
}

// NewFindSearchResults creates the use case over the index.
func NewFindSearchResults(deps FindSearchResultsDeps) *FindSearchResults {
	limits, stageLimits := deps.Limits, deps.StageLimits
	if limits == (searchquery.Limits{}) {
		limits = searchquery.DefaultLimits()
	}
	if stageLimits == (searchquery.StageLimits{}) {
		stageLimits = searchquery.DefaultStageLimits()
	}
	return &FindSearchResults{
		parser: deps.Parser, documents: deps.Documents, vectors: deps.Vectors, words: deps.Words,
		places: deps.Places, matcher: deps.Matcher, readers: deps.Readers, texts: deps.Texts,
		source: deps.Source, floors: deps.Floors, contentTypes: deps.ContentTypes,
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
func (u *FindSearchResults) Execute(ctx context.Context, v domain.Vault, r Request) (searchquery.SearchResults, error) {
	if err := ctx.Err(); err != nil {
		return searchquery.SearchResults{}, err
	}
	if r.Limit < 0 {
		return searchquery.SearchResults{}, fmt.Errorf("%w: %d", ErrNegativeLimit, r.Limit)
	}
	q, err := u.parse(r.Text)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	if hasMeaning(q.Conditions) && !u.hasModel() {
		return searchquery.SearchResults{}, searchquery.ErrNoModel
	}
	results, err := u.newPlan(ctx, v, q).execute(ctx)
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
	return u.applyStages(ctx, v, q, results, priority)
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

func (u *FindSearchResults) parse(text string) (searchquery.Query, error) {
	if err := searchquery.CheckQueryText(text, u.Limits); err != nil {
		return searchquery.Query{}, err
	}
	raw, err := u.parser.ParseQuery(text, searchquery.GetBuiltinCatalog())
	if err != nil {
		return searchquery.Query{}, err
	}
	return searchquery.Normalize(raw, u.Limits)
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

func (u *FindSearchResults) getDocument(path string, sourceKind domain.SourceKind) searchquery.Document {
	return searchquery.Document{
		Path: path, Title: searchquery.GetTitle(path, ""), SourceKind: sourceKind,
		Kind: searchquery.GetKind(sourceKind, u.contentTypes.GetContentType(path)),
	}
}

// fillDocuments sets the note title, the size and the time of change of the document of each result.
func (u *FindSearchResults) fillDocuments(ctx context.Context, v domain.Vault, results []searchquery.SearchResult) error {
	if len(results) == 0 {
		return nil
	}
	var paths []string
	for _, r := range results {
		paths = append(paths, r.Document.Path)
	}
	listed, err := u.documents.ListDocuments(ctx, v.ID, searchquery.DocumentQuery{PathPrefixes: paths})
	if err != nil {
		return err
	}
	byPath := make(map[string]searchquery.Document, len(listed))
	for _, d := range listed {
		byPath[d.Path] = d
	}
	for i, r := range results {
		d := byPath[r.Document.Path]
		results[i].Document.Title = searchquery.GetTitle(r.Document.Path, d.Title)
		results[i].Document.ModifiedAt, results[i].Document.Size = d.ModifiedAt, d.Size
	}
	return nil
}
