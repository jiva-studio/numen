package searchquery

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// EvaluateQueryDeps is what EvaluateQuery reads. A nil Source is an installation with no model, and a nil
// Texts reads the text of notes only.
type EvaluateQueryDeps struct {
	Documents    DocumentQueries
	Vectors      VectorQueries
	Places       sourceport.ChunkPlaces
	Contents     NoteContents
	Links        LinkReaders
	Functions    pluginport.SearchFunctions
	Matcher      sourceport.WordMatching
	Readers      vaultport.VaultReaders
	Texts        textdomain.Readers
	Source       searchsourceport.QuerySource
	Floors       FloorReader
	ContentTypes contenttypeport.ContentTypeTable
}

// EvaluateQuery evaluates a parsed query over the index of one vault.
type EvaluateQuery struct {
	documents    DocumentQueries
	vectors      VectorQueries
	places       sourceport.ChunkPlaces
	contents     NoteContents
	links        LinkReaders
	functions    pluginport.SearchFunctions
	matcher      sourceport.WordMatching
	readers      vaultport.VaultReaders
	texts        textdomain.Readers
	source       searchsourceport.QuerySource
	floors       FloorReader
	contentTypes contenttypeport.ContentTypeTable
}

// NewEvaluateQuery creates the use case over the index.
func NewEvaluateQuery(deps EvaluateQueryDeps) *EvaluateQuery {
	return &EvaluateQuery{
		documents: deps.Documents, vectors: deps.Vectors, places: deps.Places, contents: deps.Contents,
		links: deps.Links, functions: deps.Functions, matcher: deps.Matcher, readers: deps.Readers, texts: deps.Texts, source: deps.Source, floors: deps.Floors,
		contentTypes: deps.ContentTypes,
	}
}

// Execute returns the results of the query, bounded by the limits, with the dates of its conditions read in the
// zone. A limit exceeded returns searchquery.ErrLimitExceeded; meaning with no model searchquery.ErrNoModel.
func (u *EvaluateQuery) Execute(
	ctx context.Context, v domain.Vault, q searchquery.Query, zone *time.Location, limits searchquery.Limits, generation string,
) (searchquery.SearchResults, error) {
	if hasMeaning(q.Conditions) && !u.hasModel() {
		return searchquery.SearchResults{}, searchquery.ErrNoModel
	}
	req := u.newRequest(ctx, v, q, zone, limits)
	warns := &warnings{}
	calls := newFunctionCalls(u.functions, v, q, limits, generation)
	docs := newDocumentReader(u, req, warns, calls)
	return u.run(ctx, req, docs, warns, calls)
}

func (u *EvaluateQuery) hasModel() bool {
	return u.source != nil && u.source.Describe().IsDescribed()
}

func hasMeaning(conditions []searchquery.Condition) bool {
	for _, c := range conditions {
		if c.IsMeaning {
			return true
		}
	}
	return false
}
