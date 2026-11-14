package hostvault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

// Lookups reads the index data of the notes and the links of the vault.
type Lookups struct {
	Notes      query.GetNotes
	Types      *query.GetTypes
	NamedPaths *query.GetNamedPaths
	TypedPaths query.GetPathsOfType
	Links      query.GetLinks
	Backlinks  query.GetBacklinks
	Targets    *query.ResolveLinks

	session sessionAccess
}

func (l Lookups) GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.Notes.Execute(ctx, v, paths)
}

func (l Lookups) GetTypes(ctx context.Context, paths []string) (map[string]domain.NoteType, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.Types.Execute(ctx, v, paths)
}

func (l Lookups) GetNamedPaths(ctx context.Context, name string) ([]string, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.NamedPaths.Execute(ctx, v, name)
}

func (l Lookups) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.TypedPaths.Execute(ctx, v, noteType)
}

// GetLinks returns the links of the note at the path.
func (l Lookups) GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.Links.Execute(ctx, v, from)
}

// GetBacklinks returns the links to the note at the path.
func (l Lookups) GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	backlinks, err := l.Backlinks.Execute(ctx, v, to)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ResolvedLink, len(backlinks))
	for i, one := range backlinks {
		out[i] = one.ResolvedLink
	}
	return out, nil
}

// ResolveLinks resolves the addresses the note at the path links to.
func (l Lookups) ResolveLinks(
	ctx context.Context, from string, targets []string,
) (map[string]domain.ResolvedLink, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.Targets.Execute(ctx, v.ID, from, targets)
}
