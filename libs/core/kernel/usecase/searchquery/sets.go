package searchquery

import (
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func (p *plan) findSets(ctx context.Context) error {
	p.sets = make([]map[string]bool, len(p.query.Sets))
	hasProperty := false
	var problems []searchquery.Problem
	for i, s := range p.query.Sets {
		members, err := p.findSet(ctx, s)
		if message, isProblem := getTargetMessage(err); isProblem {
			problems = append(problems, searchquery.Problem{Position: s.Position, Message: message})
			continue
		}
		if err != nil {
			return err
		}
		p.sets[i] = members
		_, isLink := searchquery.FindLinkFunction(s.Function)
		hasProperty = hasProperty || (!isLink && s.Function != searchquery.FunctionHasHeading)
	}
	if len(problems) > 0 {
		return &searchquery.ParseError{Problems: problems}
	}
	if !hasProperty {
		return nil
	}
	count, err := p.u.index.contents.CountUnreadableFrontmatter(ctx, p.v.ID)
	if err != nil {
		return err
	}
	if count > 0 {
		p.warnings = append(p.warnings, fmt.Sprintf("%d notes have frontmatter that cannot be read; property() and hasProperty() treat them as having no keys", count))
	}
	return nil
}

func (p *plan) findSet(ctx context.Context, s searchquery.SetCondition) (map[string]bool, error) {
	if _, isLink := searchquery.FindLinkFunction(s.Function); isLink {
		if p.u.links == nil {
			return nil, ErrNoLinks
		}
		return p.findLinkSet(ctx, s)
	}
	members := map[string]bool{}
	if s.Function == searchquery.FunctionHasHeading {
		headings, err := p.u.index.contents.FindHeadings(ctx, p.v.ID, s.Text)
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
	entries, err := p.u.index.contents.ListPropertyValues(ctx, p.v.ID, s.Key)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if searchquery.IsPropertyMatch(e.Value, s.Comparison, s.Operands, p.zone) {
			members[e.Path] = true
		}
	}
	return members, nil
}

func (p *plan) getMembership(path string) []bool {
	if len(p.sets) == 0 {
		return nil
	}
	membership := make([]bool, len(p.sets))
	for i, members := range p.sets {
		membership[i] = members[path]
	}
	return membership
}

func (p *plan) getSetPaths(indexes []int) ([]string, bool) {
	var smallest map[string]bool
	for _, index := range indexes {
		if members := p.sets[index]; smallest == nil || len(members) < len(smallest) {
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

func (p *plan) loadDocuments(ctx context.Context, paths []string) error {
	if !hasCompareLeaf(p.query.Root) {
		return nil
	}
	var missing []string
	seen := map[string]bool{}
	for _, path := range paths {
		if _, found := p.documents[path]; !found && !seen[path] {
			seen[path] = true
			missing = append(missing, path)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	listed, err := p.u.index.documents.ListDocuments(ctx, p.v.ID, searchquery.DocumentQuery{Paths: missing, IsWithinPaths: true})
	if err != nil {
		return err
	}
	for _, d := range listed {
		p.fillKind(d)
	}
	return nil
}

func (p *plan) getDocument(path string, sourceKind domain.SourceKind) searchquery.Document {
	if d, found := p.documents[path]; found {
		return d
	}
	d := p.u.getDocument(path, sourceKind)
	d.IsInSet = p.getMembership(path)
	return d
}

func hasCompareLeaf(node searchquery.Node) bool {
	if node.Operator == searchquery.OperatorLeaf {
		return node.Leaf.Kind == searchquery.LeafCompare
	}
	return slices.ContainsFunc(node.Children, hasCompareLeaf)
}

func (p *plan) getScopePaths(s searchquery.Scope) ([]string, bool) {
	paths, isWithinPaths := p.getSetPaths(s.SetIndexes)
	switch {
	case !s.IsWithinPaths:
		return paths, isWithinPaths
	case !isWithinPaths:
		return s.Paths, true
	}
	found := make(map[string]bool, len(s.Paths))
	for _, path := range s.Paths {
		found[path] = true
	}
	return slices.DeleteFunc(paths, func(path string) bool { return !found[path] }), true
}
