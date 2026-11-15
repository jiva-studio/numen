package searchquery

import (
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func (p *plan) listScopeDocuments(ctx context.Context, s searchquery.Scope) ([]searchquery.Document, bool, error) {
	if s.IsEveryDocument() {
		return nil, false, nil
	}
	paths, isWithinPaths := p.getScopePaths(s)
	if isWithinPaths && len(paths) == 0 {
		return nil, true, nil
	}
	listed, err := p.u.index.documents.ListDocuments(ctx, p.v.ID, searchquery.DocumentQuery{
		PathPrefixes: s.PathPrefixes, SourceKinds: s.SourceKinds, Paths: paths, IsWithinPaths: isWithinPaths,
		Recipe: p.recipe, Limit: p.u.Limits.MaxDocuments,
	})
	if err != nil {
		return nil, false, err
	}
	if len(listed) > p.u.Limits.MaxDocuments {
		return nil, false, nil
	}
	if err := p.matchCalls(ctx, getPaths(listed)); err != nil {
		return nil, false, err
	}
	var documents []searchquery.Document
	for _, d := range listed {
		if d = p.fillKind(d); s.IsDocumentMatch(d) {
			documents = append(documents, d)
		}
	}
	return documents, true, nil
}

func (p *plan) listCandidateDocuments(ctx context.Context) ([]searchquery.Document, error) {
	if !hasPositiveDocumentLeaf(p.query.Root) {
		return nil, nil
	}
	s := searchquery.GetDocumentScope(p.query, p.calls.found)
	paths, isWithinPaths := p.getScopePaths(s)
	if isWithinPaths && len(paths) == 0 {
		return nil, nil
	}
	listed, err := p.u.index.documents.ListDocuments(ctx, p.v.ID, searchquery.DocumentQuery{
		PathPrefixes: s.PathPrefixes, SourceKinds: s.SourceKinds, Paths: paths, IsWithinPaths: isWithinPaths, Recipe: p.recipe,
	})
	if err != nil {
		return nil, err
	}
	if err := p.matchCalls(ctx, getPaths(listed)); err != nil {
		return nil, err
	}
	var documents []searchquery.Document
	for _, d := range listed {
		if d = p.fillKind(d); searchquery.IsDocumentMatch(p.query.Root, d) {
			documents = append(documents, d)
		}
	}
	if len(documents) > p.u.Limits.MaxDocuments {
		return nil, fmt.Errorf(
			"%w: the document conditions match more than %d documents (MaxDocuments); add a meaning or words condition or narrow the path",
			searchquery.ErrLimitExceeded, p.u.Limits.MaxDocuments)
	}
	return documents, nil
}

func (p *plan) fillKind(d searchquery.Document) searchquery.Document {
	d.Kind = searchquery.GetKind(d.SourceKind, p.u.contentTypes.GetContentType(d.Path))
	d.Title = searchquery.GetTitle(d.Path, d.Title)
	d.IsInSet = p.getMembership(d.Path)
	p.documents[d.Path] = d
	return p.withCalls(d)
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
