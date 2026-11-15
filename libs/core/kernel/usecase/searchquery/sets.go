package searchquery

import (
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

func (r *documentReader) findSets(ctx context.Context) error {
	r.sets = make([]map[string]bool, len(r.req.Query.Sets))
	hasProperty := false
	var problems []searchquery.Problem
	for i, s := range r.req.Query.Sets {
		members, err := r.findSet(ctx, s)
		if message, isProblem := getTargetMessage(err); isProblem {
			problems = append(problems, searchquery.Problem{Position: s.Position, Message: message})
			continue
		}
		if err != nil {
			return err
		}
		r.sets[i] = members
		_, isLink := searchquery.FindLinkFunction(s.Function)
		hasProperty = hasProperty || (!isLink && s.Function != searchquery.FunctionHasHeading)
	}
	if len(problems) > 0 {
		return &searchquery.ParseError{Problems: problems}
	}
	if !hasProperty {
		return nil
	}
	count, err := r.contents.CountUnreadableFrontmatter(ctx, r.req.Vault.ID)
	if err != nil {
		return err
	}
	if count > 0 {
		r.warnings.add(fmt.Sprintf("%d notes have frontmatter that cannot be read; property() and hasProperty() treat them as having no keys", count))
	}
	return nil
}

func (r *documentReader) findSet(ctx context.Context, s searchquery.SetCondition) (map[string]bool, error) {
	if _, isLink := searchquery.FindLinkFunction(s.Function); isLink {
		if r.links == nil {
			return nil, ErrNoLinks
		}
		return r.links.findLinkSet(ctx, s)
	}
	members := map[string]bool{}
	if s.Function == searchquery.FunctionHasHeading {
		headings, err := r.contents.FindHeadings(ctx, r.req.Vault.ID, s.Text)
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
	entries, err := r.contents.ListPropertyValues(ctx, r.req.Vault.ID, s.Key)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if searchquery.IsPropertyMatch(e.Value, s.Comparison, s.Operands, r.req.Zone) {
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
	if !hasCompareLeaf(r.req.Query.Root) {
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
	listed, err := r.lister.ListDocuments(ctx, r.req.Vault.ID, searchquery.DocumentQuery{Paths: missing})
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
