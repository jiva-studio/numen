package searchquery

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// EvaluateQueryDeps is what EvaluateQuery reads. A nil Source is an installation with no model, and a nil
// Texts reads the text of notes only.
type EvaluateQueryDeps struct {
	Chunks       ChunkIndex
	Contents     NoteContents
	Links        LinkReaders
	Matcher      sourceport.WordMatching
	Readers      vaultport.VaultReaders
	Texts        textdomain.Readers
	Source       searchsourceport.QuerySource
	Floors       FloorReader
	ContentTypes contenttypeport.ContentTypeTable
}

// EvaluateQuery evaluates a parsed query over the index of one vault.
type EvaluateQuery struct {
	chunks       ChunkIndex
	contents     NoteContents
	links        LinkReaders
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
		chunks: deps.Chunks, contents: deps.Contents, links: deps.Links, matcher: deps.Matcher, readers: deps.Readers,
		texts: deps.Texts, source: deps.Source, floors: deps.Floors, contentTypes: deps.ContentTypes,
	}
}

// Execute returns the results of the query, bounded by the limits, with the dates of its conditions read in the
// zone. A limit exceeded returns searchquery.ErrLimitExceeded; meaning with no model searchquery.ErrNoModel.
func (u *EvaluateQuery) Execute(
	ctx context.Context, v domain.Vault, q searchquery.Query, zone *time.Location, limits searchquery.Limits,
) (searchquery.SearchResults, error) {
	if hasMeaning(q.Conditions) && !u.hasModel() {
		return searchquery.SearchResults{}, searchquery.ErrNoModel
	}
	req := u.newRequest(ctx, v, q, zone, limits)
	warns := &warnings{}
	docs := newDocumentReader(u.chunks, u.contents, u.links, u.contentTypes, req, warns)
	return u.run(ctx, req, docs, warns)
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
