package searchquery

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrNoLinks is a query with a link function in an installation that reads no links.
var ErrNoLinks = errors.New("the query uses a link function, and the installation reads no links")

const messageNotNote = "the target of %s is not a note: %s"

// target is what the first argument of a link function names: a note, an indexed file, or an address that
// is neither (a URL).
type target struct {
	path    string
	address string
	isNote  bool
}

func (p *plan) computeSets(ctx context.Context) error {
	if len(p.query.Sets) == 0 {
		return nil
	}
	if p.u.links == nil {
		return ErrNoLinks
	}
	p.sets = make([]map[string]bool, len(p.query.Sets))
	var problems []searchquery.Problem
	for i, s := range p.query.Sets {
		set, err := p.computeSet(ctx, s)
		if message, isProblem := getTargetMessage(err); isProblem {
			problems = append(problems, searchquery.Problem{Position: s.Position, Message: message})
			continue
		}
		if err != nil {
			return err
		}
		p.sets[i] = set
	}
	if len(problems) > 0 {
		return &searchquery.ParseError{Problems: problems}
	}
	return nil
}

type targetError struct{ message string }

func (e *targetError) Error() string { return e.message }

func getTargetMessage(err error) (string, bool) {
	if ambiguous, ok := errors.AsType[*searchquery.AmbiguousTargetError](err); ok {
		return ambiguous.Error(), true
	}
	if problem, ok := errors.AsType[*targetError](err); ok {
		return problem.message, true
	}
	if errors.Is(err, searchquery.ErrNoTarget) {
		return err.Error(), true
	}
	return "", false
}

func (p *plan) computeSet(ctx context.Context, s searchquery.SetCondition) (map[string]bool, error) {
	f, _ := searchquery.FindLinkFunction(s.Function)
	t, err := p.resolveTarget(ctx, s.Target)
	if err != nil {
		return nil, err
	}
	if f.Direction == searchquery.DirectionTo {
		return p.findLinksTo(ctx, t, f.Role, s.LinkType)
	}
	if !t.isNote {
		if t.address != "" {
			return nil, &targetError{message: fmt.Sprintf(messageNotNote, s.Function, t.address)}
		}
		return map[string]bool{}, nil
	}
	if f.Direction == searchquery.DirectionFrom {
		return p.findLinksFrom(ctx, t.path, f.Role, s.LinkType)
	}
	return p.findRelatives(ctx, t.path, f, s.LinkType)
}

func (p *plan) resolveTarget(ctx context.Context, x string) (target, error) {
	a := domain.ParseAddress(x)
	switch a.Scheme {
	case domain.SchemeNote:
		paths, err := p.u.links.GetNoteCandidates(ctx, p.v.ID, a)
		if err != nil {
			return target{}, err
		}
		if len(paths) == 0 {
			return target{}, fmt.Errorf("%w: no note with identifier %s in this vault", searchquery.ErrNoTarget, a.Value)
		}
		return target{path: paths[0], isNote: true}, nil
	case domain.SchemeName:
		candidates, err := p.u.links.GetNoteCandidates(ctx, p.v.ID, a)
		if err != nil {
			return target{}, err
		}
		chosen, err := searchquery.ChooseTarget(a.Value, candidates)
		if err != nil || chosen != "" {
			return target{path: chosen, isNote: true}, err
		}
		return p.resolveFile(ctx, x, a.Value)
	case domain.SchemeAsset:
		return p.resolveFile(ctx, x, a.Value)
	}
	return target{address: a.String()}, nil
}

func (p *plan) resolveFile(ctx context.Context, x, value string) (target, error) {
	files, err := p.getFiles(ctx)
	if err != nil {
		return target{}, err
	}
	filePath := searchquery.CleanPath(value)
	chosen, err := searchquery.ChooseTarget(filePath, files[searchquery.GetFileName(filePath)])
	if err != nil {
		return target{}, err
	}
	if chosen == "" {
		return target{}, fmt.Errorf("%w: %s", searchquery.ErrNoTarget, x)
	}
	return target{path: chosen}, nil
}

func (p *plan) getFiles(ctx context.Context) (map[string][]string, error) {
	if p.files != nil {
		return p.files, nil
	}
	paths, err := p.u.links.ListFilePaths(ctx, p.v.ID)
	if err != nil {
		return nil, err
	}
	p.files = searchquery.GroupFilesByName(paths)
	return p.files, nil
}

func (p *plan) findLinksTo(ctx context.Context, t target, role domain.LinkRole, linkType string) (map[string]bool, error) {
	set := map[string]bool{}
	if t.isNote {
		backlinks, err := p.u.links.GetBacklinksOf(ctx, p.v.ID, []string{t.path})
		if err != nil {
			return nil, err
		}
		for _, l := range backlinks[t.path] {
			if searchquery.IsLinkMatch(l, role, linkType) {
				set[l.From] = true
			}
		}
		return set, nil
	}
	value := t.path
	if t.address != "" {
		value = domain.ParseAddress(t.address).Value
	}
	links, err := p.u.links.ListLinksByName(ctx, p.v.ID, domain.FoldName(domain.LinkName(value)))
	if err != nil {
		return nil, err
	}
	files, err := p.getFiles(ctx)
	if err != nil {
		return nil, err
	}
	for _, l := range links {
		if searchquery.IsLinkMatch(l, role, linkType) && isLinkToFile(l, t, files) {
			set[l.From] = true
		}
	}
	return set, nil
}

func isLinkToFile(l domain.ResolvedLink, t target, files map[string][]string) bool {
	if t.address != "" {
		return l.Target.String() == t.address
	}
	return searchquery.ChooseFile(l.From, l.Target, files) == t.path
}

func (p *plan) findLinksFrom(ctx context.Context, from string, role domain.LinkRole, linkType string) (map[string]bool, error) {
	links, err := p.u.links.GetLinksOf(ctx, p.v.ID, []string{from})
	if err != nil {
		return nil, err
	}
	files, err := p.getFiles(ctx)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, l := range links[from] {
		switch {
		case !searchquery.IsLinkMatch(l, role, linkType):
		case l.To != "":
			if _, isCrossed := l.InVault(p.v.ID); !isCrossed {
				set[l.To] = true
			}
		default:
			if filePath := searchquery.ChooseFile(l.From, l.Target, files); filePath != "" {
				set[filePath] = true
			}
		}
	}
	return set, nil
}

func (p *plan) findRelatives(ctx context.Context, start string, f searchquery.LinkFunction, linkType string) (map[string]bool, error) {
	visited := map[string]bool{start: true}
	set := map[string]bool{}
	frontier := []string{start}
	for step := 1; len(frontier) > 0; step++ {
		relatives, err := p.getRelatives(ctx, frontier, f.Direction, linkType)
		if err != nil {
			return nil, err
		}
		fresh := markVisited(visited, relatives)
		if err := p.checkHierarchy(start, step, len(fresh), len(visited)); err != nil {
			return nil, err
		}
		for _, r := range fresh {
			set[r] = true
		}
		if !f.IsTransitive {
			break
		}
		frontier = fresh
	}
	return set, nil
}

func markVisited(visited map[string]bool, relatives []string) []string {
	var fresh []string
	for _, r := range relatives {
		if !visited[r] {
			visited[r] = true
			fresh = append(fresh, r)
		}
	}
	return fresh
}

func (p *plan) checkHierarchy(start string, step, freshCount, visitedCount int) error {
	if freshCount > 0 && step > p.u.Limits.MaxLinkDepth {
		return fmt.Errorf("%w: the hierarchy of %s is deeper than %d steps (MaxLinkDepth)",
			searchquery.ErrLimitExceeded, start, p.u.Limits.MaxLinkDepth)
	}
	if visitedCount > p.u.Limits.MaxDocuments {
		return fmt.Errorf("%w: the hierarchy of %s holds more than %d notes (MaxDocuments)",
			searchquery.ErrLimitExceeded, start, p.u.Limits.MaxDocuments)
	}
	return nil
}

func (p *plan) getRelatives(ctx context.Context, notes []string, direction searchquery.LinkDirection, linkType string) ([]string, error) {
	links, err := p.u.links.GetLinksOf(ctx, p.v.ID, notes)
	if err != nil {
		return nil, err
	}
	backlinks, err := p.u.links.GetBacklinksOf(ctx, p.v.ID, notes)
	if err != nil {
		return nil, err
	}
	return searchquery.GetRelatives(direction, linkType, p.v.ID, links, backlinks), nil
}

func (p *plan) markSets(d searchquery.Document) searchquery.Document {
	if len(p.sets) == 0 {
		return d
	}
	d.IsInSet = make([]bool, len(p.sets))
	for i, set := range p.sets {
		d.IsInSet[i] = set[d.Path]
	}
	return d
}

func (p *plan) getScopePaths(indexes []int) ([]string, bool) {
	var smallest map[string]bool
	for _, i := range indexes {
		if i < len(p.sets) && (smallest == nil || len(p.sets[i]) < len(smallest)) {
			smallest = p.sets[i]
		}
	}
	if smallest == nil {
		return nil, false
	}
	return slices.Sorted(maps.Keys(smallest)), true
}
