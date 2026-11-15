package searchquery

import (
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/note/domain"
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
}

func newDocumentReader(
	lister DocumentQueries, contents NoteContents, links LinkReaders, contentTypes contenttypeport.ContentTypeTable, req *searchquery.Evaluation,
	warns *warnings,
) *documentReader {
	reader := &documentReader{
		lister: lister, contents: contents, contentTypes: contentTypes, req: req, warnings: warns, documents: map[string]searchquery.Document{},
	}
	if links != nil {
		reader.links = &linkFinder{links: links, req: req}
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
	s := searchquery.GetDocumentScope(r.req.Query)
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
	return d
}

func (r *documentReader) getDocument(path string, sourceKind domain.SourceKind) searchquery.Document {
	if d, found := r.documents[path]; found {
		return d
	}
	d := searchquery.Document{
		Path: path, Title: searchquery.GetTitle(path, ""), SourceKind: sourceKind,
		Kind: searchquery.GetKind(sourceKind, r.contentTypes.GetContentType(path)),
	}
	d.IsInSet = r.getMembership(path)
	return d
}

// FillTitles sets the title of the document of each note result.
func (u *EvaluateQuery) FillTitles(ctx context.Context, v domain.Vault, results []searchquery.SearchResult) error {
	var paths []string
	for _, one := range results {
		if one.Document.SourceKind == domain.KindNote {
			paths = append(paths, one.Document.Path)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	listed, err := u.documents.ListDocuments(ctx, v.ID, searchquery.DocumentQuery{PathPrefixes: paths})
	if err != nil {
		return err
	}
	titles := make(map[string]string, len(listed))
	for _, d := range listed {
		titles[d.Path] = d.Title
	}
	for i, one := range results {
		results[i].Document.Title = searchquery.GetTitle(one.Document.Path, titles[one.Document.Path])
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
