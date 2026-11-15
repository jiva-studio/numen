package searchquery

import (
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func (r *documentReader) findSets(ctx context.Context) error {
	r.sets = make([]map[string]bool, len(r.req.query.Sets))
	hasProperty := false
	for i, s := range r.req.query.Sets {
		members, err := r.findSet(ctx, s)
		if err != nil {
			return err
		}
		r.sets[i] = members
		hasProperty = hasProperty || s.Function != searchquery.FunctionHasHeading
	}
	if !hasProperty {
		return nil
	}
	count, err := r.index.contents.CountUnreadableFrontmatter(ctx, r.req.vault.ID)
	if err != nil {
		return err
	}
	if count > 0 {
		r.warnings.add(fmt.Sprintf("%d notes have frontmatter that cannot be read; property() and hasProperty() treat them as having no keys", count))
	}
	return nil
}

func (r *documentReader) findSet(ctx context.Context, s searchquery.SetCondition) (map[string]bool, error) {
	members := map[string]bool{}
	if s.Function == searchquery.FunctionHasHeading {
		headings, err := r.index.contents.FindHeadings(ctx, r.req.vault.ID, s.Text)
		if err != nil {
			return nil, err
		}
		for _, h := range headings {
			if searchquery.IsHeadingMatch(h.Text, s.Text) {
				members[h.Path] = true
			}
		}
		return members, nil
	}
	entries, err := r.index.contents.ListPropertyValues(ctx, r.req.vault.ID, s.Key)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if searchquery.IsPropertyMatch(e.Value, s.Comparison, s.Operands, r.req.zone) {
			members[e.Path] = true
		}
	}
	return members, nil
}

func (r *documentReader) getMembership(path string) []bool {
	if len(r.sets) == 0 {
		return nil
	}
	membership := make([]bool, len(r.sets))
	for i, members := range r.sets {
		membership[i] = members[path]
	}
	return membership
}

func (r *documentReader) getSetPaths(indexes []int) ([]string, bool) {
	var smallest map[string]bool
	for _, index := range indexes {
		if members := r.sets[index]; smallest == nil || len(members) < len(smallest) {
			smallest = members
		}
	}
	if smallest == nil {
		return nil, false
	}
	paths := make([]string, 0, len(smallest))
	for path := range smallest {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths, true
}

func (r *documentReader) loadDocuments(ctx context.Context, paths []string) error {
	if !hasCompareLeaf(r.req.query.Root) {
		return nil
	}
	var missing []string
	seen := map[string]bool{}
	for _, path := range paths {
		if _, found := r.documents[path]; !found && !seen[path] {
			seen[path] = true
			missing = append(missing, path)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	listed, err := r.index.documents.ListDocuments(ctx, r.req.vault.ID, searchquery.DocumentQuery{Paths: missing})
	if err != nil {
		return err
	}
	for _, d := range listed {
		r.fillKind(d)
	}
	return nil
}

func hasCompareLeaf(node searchquery.Node) bool {
	if node.Operator == searchquery.OperatorLeaf {
		return node.Leaf.Kind == searchquery.LeafCompare
	}
	return slices.ContainsFunc(node.Children, hasCompareLeaf)
}
