package searchquery

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

// listScopeDocuments returns the documents of the scope, and false when the scope is every document or holds
// more than Limits.MaxDocuments of them: the searches then run over the vault and the scope filters after.
func (p *plan) listScopeDocuments(ctx context.Context, s searchquery.Scope) ([]searchquery.Document, bool, error) {
	if s.IsEveryDocument() {
		return nil, false, nil
	}
	listed, err := p.u.documents.ListDocuments(ctx, p.v.ID, searchquery.DocumentQuery{
		PathPrefixes: s.PathPrefixes, SourceKinds: s.SourceKinds, Recipe: p.recipe, Limit: p.u.Limits.MaxDocuments,
	})
	if err != nil {
		return nil, false, err
	}
	if len(listed) > p.u.Limits.MaxDocuments {
		return nil, false, nil
	}
	var documents []searchquery.Document
	for _, d := range listed {
		if d = p.fillKind(d); s.IsDocumentMatch(d) {
			documents = append(documents, d)
		}
	}
	return documents, true, nil
}

// listCandidateDocuments returns the documents the query matches as whole files when it has a document
// condition that is not negated.
func (p *plan) listCandidateDocuments(ctx context.Context) ([]searchquery.Document, error) {
	if !hasPositiveDocumentLeaf(p.query.Root) {
		return nil, nil
	}
	s := searchquery.GetDocumentScope(p.query)
	listed, err := p.u.documents.ListDocuments(ctx, p.v.ID, searchquery.DocumentQuery{
		PathPrefixes: s.PathPrefixes, SourceKinds: s.SourceKinds, Recipe: p.recipe,
	})
	if err != nil {
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
	return d
}

func hasPositiveDocumentLeaf(node searchquery.Node) bool {
	if node.Operator == searchquery.OperatorLeaf {
		return !node.Leaf.IsChunkLevel() && !node.IsNegated
	}
	for _, child := range node.Children {
		if hasPositiveDocumentLeaf(child) {
			return true
		}
	}
	return false
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
