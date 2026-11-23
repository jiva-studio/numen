package searchquery

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
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

func (l *linkFinder) findLinkSet(ctx context.Context, s searchquery.SetCondition) (map[string]bool, error) {
	f, _ := searchquery.FindLinkFunction(s.Function)
	t, err := l.resolveTarget(ctx, s.Target)
	if err != nil {
		return nil, err
	}
	if f.Direction == searchquery.DirectionTo {
		return l.findLinksTo(ctx, t, f.Role, s.LinkType)
	}
	if !t.isNote {
		if t.address != "" {
			return nil, &targetError{message: fmt.Sprintf(messageNotNote, s.Function, t.address)}
		}
		return map[string]bool{}, nil
	}
	if f.Direction == searchquery.DirectionFrom {
		return l.findLinksFrom(ctx, t.path, f.Role, s.LinkType)
	}
	return l.findRelatives(ctx, t.path, f, s.LinkType)
}

func (l *linkFinder) resolveTarget(ctx context.Context, x string) (target, error) {
	a := domain.ParseAddress(x)
	switch a.Scheme {
	case domain.SchemeNote:
		paths, err := l.links.GetNoteCandidates(ctx, l.req.Vault.ID, a)
		if err != nil {
			return target{}, err
		}
		if len(paths) == 0 {
			return target{}, fmt.Errorf("%w: no note with identifier %s in this vault", searchquery.ErrNoTarget, a.Value)
		}
		return target{path: paths[0], isNote: true}, nil
	case domain.SchemeName:
		candidates, err := l.links.GetNoteCandidates(ctx, l.req.Vault.ID, a)
		if err != nil {
			return target{}, err
		}
		chosen, err := searchquery.ChooseTarget(a.Value, candidates)
		if err != nil || chosen != "" {
			return target{path: chosen, isNote: true}, err
		}
		return l.resolveFile(ctx, x, a.Value)
	case domain.SchemeAsset:
		return l.resolveFile(ctx, x, a.Value)
	}
	return target{address: a.String()}, nil
}

func (l *linkFinder) resolveFile(ctx context.Context, x, value string) (target, error) {
	files, err := l.getFiles(ctx)
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

func (l *linkFinder) getFiles(ctx context.Context) (map[string][]string, error) {
	if l.files != nil {
		return l.files, nil
	}
	paths, err := l.links.ListFilePaths(ctx, l.req.Vault.ID)
	if err != nil {
		return nil, err
	}
	l.files = searchquery.GroupFilesByName(paths)
	return l.files, nil
}

func (l *linkFinder) findLinksTo(ctx context.Context, t target, role domain.LinkRole, linkType string) (map[string]bool, error) {
	set := map[string]bool{}
	if t.isNote {
		backlinks, err := l.links.ListBacklinksOf(ctx, l.req.Vault.ID, []string{t.path})
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
	links, err := l.links.ListLinksByName(ctx, l.req.Vault.ID, domain.NormalizeName(domain.LinkName(value)))
	if err != nil {
		return nil, err
	}
	files, err := l.getFiles(ctx)
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

func (l *linkFinder) findLinksFrom(ctx context.Context, from string, role domain.LinkRole, linkType string) (map[string]bool, error) {
	links, err := l.links.ListLinksOf(ctx, l.req.Vault.ID, []string{from})
	if err != nil {
		return nil, err
	}
	files, err := l.getFiles(ctx)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, link := range links[from] {
		switch {
		case !searchquery.IsLinkMatch(link, role, linkType):
		case link.To != "":
			if _, isCrossed := link.ResolveVault(l.req.Vault.ID); !isCrossed {
				set[link.To] = true
			}
		default:
			if filePath := searchquery.ChooseFile(link.From, link.Target, files); filePath != "" {
				set[filePath] = true
			}
		}
	}
	return set, nil
}

func (l *linkFinder) findRelatives(ctx context.Context, start string, f searchquery.LinkFunction, linkType string) (map[string]bool, error) {
	visited := map[string]bool{start: true}
	set := map[string]bool{}
	frontier := []string{start}
	for step := 1; len(frontier) > 0; step++ {
		relatives, err := l.getRelatives(ctx, frontier, f.Direction, linkType)
		if err != nil {
			return nil, err
		}
		fresh := markVisited(visited, relatives)
		if err := l.checkHierarchy(start, step, len(fresh), len(visited)); err != nil {
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

func (l *linkFinder) checkHierarchy(start string, step, freshCount, visitedCount int) error {
	if freshCount > 0 && step > l.req.Limits.MaxLinkDepth {
		return fmt.Errorf("%w: the hierarchy of %s is deeper than %d steps (MaxLinkDepth)",
			searchquery.ErrLimitExceeded, start, l.req.Limits.MaxLinkDepth)
	}
	if visitedCount > l.req.Limits.MaxDocuments {
		return fmt.Errorf("%w: the hierarchy of %s holds more than %d notes (MaxDocuments)",
			searchquery.ErrLimitExceeded, start, l.req.Limits.MaxDocuments)
	}
	return nil
}

func (l *linkFinder) getRelatives(ctx context.Context, notes []string, direction searchquery.LinkDirection, linkType string) ([]string, error) {
	links, err := l.links.ListLinksOf(ctx, l.req.Vault.ID, notes)
	if err != nil {
		return nil, err
	}
	backlinks, err := l.links.ListBacklinksOf(ctx, l.req.Vault.ID, notes)
	if err != nil {
		return nil, err
	}
	return searchquery.GetRelatives(direction, linkType, l.req.Vault.ID, links, backlinks), nil
}

type linkFinder struct {
	links LinkReaders
	req   *searchquery.Evaluation
	files map[string][]string
}
