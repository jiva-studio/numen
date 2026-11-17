package searchquery

import (
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type documentReader struct {
	lister       DocumentQueries
	contents     NoteContents
	contentTypes contenttypeport.ContentTypeTable
	req          *searchquery.Evaluation
	warnings     *warnings
	sets         []map[string]bool
	documents    map[string]searchquery.Document
	links        *linkFinder
	calls        documentCalls
}

// documentCalls marks the calls of the plugin functions a document matches.
type documentCalls interface {
	withCalls(d searchquery.Document) searchquery.Document
	match(ctx context.Context, paths []string) error
}

func newDocumentReader(u *EvaluateQuery, req *searchquery.Evaluation, warns *warnings, calls documentCalls) *documentReader {
	reader := &documentReader{
		lister: u.documents, contents: u.contents, contentTypes: u.contentTypes, req: req, warnings: warns, calls: calls,
		documents: map[string]searchquery.Document{},
	}
	if u.links != nil {
		reader.links = &linkFinder{links: u.links, req: req}
	}
	return reader
}

func (r *documentReader) listScopeDocuments(ctx context.Context, s searchquery.Scope) ([]searchquery.Document, bool, error) {
	if s.IsEveryDocument() {
		return nil, false, nil
	}
	paths, isNarrowed := r.getSetPaths(s.SetIndexes)
	if isNarrowed && len(paths) == 0 {
		return nil, true, nil
	}
	listed, err := r.lister.ListDocuments(ctx, r.req.Vault.ID, searchquery.DocumentQuery{
		PathPrefixes: s.PathPrefixes, SourceKinds: s.SourceKinds, Paths: paths, Recipe: r.req.Recipe, Limit: r.req.Limits.MaxDocuments,
	})
	if err != nil {
		return nil, false, err
	}
	if len(listed) > r.req.Limits.MaxDocuments {
		return nil, false, nil
	}
	if err := r.calls.match(ctx, getPaths(listed)); err != nil {
		return nil, false, err
	}
	var documents []searchquery.Document
	for _, d := range listed {
		if d = r.fillKind(d); s.IsDocumentMatch(d) {
			documents = append(documents, d)
		}
	}
	return documents, true, nil
}

func (r *documentReader) listCandidateDocuments(ctx context.Context) ([]searchquery.Document, error) {
	if !hasPositiveDocumentLeaf(r.req.Query.Root) {
		return nil, nil
	}
	s := searchquery.GetDocumentScope(r.req.Query, r.req.Found)
	paths, isNarrowed := r.getSetPaths(s.SetIndexes)
	if isNarrowed && len(paths) == 0 {
		return nil, nil
	}
	listed, err := r.lister.ListDocuments(ctx, r.req.Vault.ID, searchquery.DocumentQuery{
		PathPrefixes: s.PathPrefixes, SourceKinds: s.SourceKinds, Paths: paths, Recipe: r.req.Recipe,
	})
	if err != nil {
		return nil, err
	}
	if err := r.calls.match(ctx, getPaths(listed)); err != nil {
		return nil, err
	}
	var documents []searchquery.Document
	for _, d := range listed {
		if d = r.fillKind(d); searchquery.IsDocumentMatch(r.req.Query.Root, d) {
			documents = append(documents, d)
		}
	}
	if len(documents) > r.req.Limits.MaxDocuments {
		return nil, fmt.Errorf(
			"%w: the document conditions match more than %d documents (MaxDocuments); add a meaning or words condition or narrow the path",
			searchquery.ErrLimitExceeded, r.req.Limits.MaxDocuments)
	}
	return documents, nil
}

func (r *documentReader) fillKind(d searchquery.Document) searchquery.Document {
	d.Kind = searchquery.GetKind(d.SourceKind, r.contentTypes.GetContentType(d.Path))
	d.Title = searchquery.GetTitle(d.Path, d.Title)
	d.IsInSet = r.getMembership(d.Path)
	r.documents[d.Path] = d
	return r.calls.withCalls(d)
}

func (r *documentReader) getDocument(path string, sourceKind domain.SourceKind) searchquery.Document {
	if d, found := r.documents[path]; found {
		return r.calls.withCalls(d)
	}
	d := searchquery.Document{
		Path: path, Title: searchquery.GetTitle(path, ""), SourceKind: sourceKind,
		Kind: searchquery.GetKind(sourceKind, r.contentTypes.GetContentType(path)),
	}
	d.IsInSet = r.getMembership(path)
	return r.calls.withCalls(d)
}

// FillDocuments sets the title, the time of change and the size of the document of each result.
func (u *EvaluateQuery) FillDocuments(ctx context.Context, v domain.Vault, results []searchquery.SearchResult) error {
	if len(results) == 0 {
		return nil
	}
	var paths []string
	for _, one := range results {
		paths = append(paths, one.Document.Path)
	}
	listed, err := u.documents.ListDocuments(ctx, v.ID, searchquery.DocumentQuery{PathPrefixes: paths})
	if err != nil {
		return err
	}
	byPath := make(map[string]searchquery.Document, len(listed))
	for _, d := range listed {
		byPath[d.Path] = d
	}
	for i, one := range results {
		d := byPath[one.Document.Path]
		results[i].Document.Title = searchquery.GetTitle(one.Document.Path, d.Title)
		results[i].Document.ModifiedAt, results[i].Document.Size = d.ModifiedAt, d.Size
	}
	return nil
}

func hasPositiveDocumentLeaf(node searchquery.Node) bool {
	if node.Operator == searchquery.OperatorLeaf {
		return !node.Leaf.IsChunkLevel() && !node.IsNegated
	}
	return slices.ContainsFunc(node.Children, hasPositiveDocumentLeaf)
}

func getChunkTotal(documents []searchquery.Document) int {
	total := 0
	for _, d := range documents {
		total += d.ChunkCount
	}
	return total
}

func getPaths(documents []searchquery.Document) []string {
	paths := make([]string, len(documents))
	for i, d := range documents {
		paths[i] = d.Path
	}
	return paths
}
