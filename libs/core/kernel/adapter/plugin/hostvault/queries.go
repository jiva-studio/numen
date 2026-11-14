package hostvault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

// Notes reads the index data of the notes of the vault.
type Notes struct {
	Notes      query.GetNotes
	Types      *query.GetTypes
	NamedPaths *query.GetNamedPaths
	TypedPaths query.GetPathsOfType

	session sessionAccess
}

func (n Notes) GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error) {
	v, err := n.session.open()
	if err != nil {
		return nil, err
	}
	return n.Notes.Execute(ctx, v, paths)
}

func (n Notes) GetTypes(ctx context.Context, paths []string) (map[string]domain.NoteType, error) {
	v, err := n.session.open()
	if err != nil {
		return nil, err
	}
	return n.Types.Execute(ctx, v, paths)
}

func (n Notes) GetNamedPaths(ctx context.Context, name string) ([]string, error) {
	v, err := n.session.open()
	if err != nil {
		return nil, err
	}
	return n.NamedPaths.Execute(ctx, v, name)
}

func (n Notes) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	v, err := n.session.open()
	if err != nil {
		return nil, err
	}
	return n.TypedPaths.Execute(ctx, v, noteType)
}

// Links reads the links between the notes of the vault.
type Links struct {
	Links     query.GetLinks
	Backlinks query.GetBacklinks
	Targets   *query.ResolveLinks

	session sessionAccess
}

// GetLinks returns the links of the note at the path.
func (l Links) GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.Links.Execute(ctx, v, from)
}

// GetBacklinks returns the links to the note at the path.
func (l Links) GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error) {
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
func (l Links) ResolveLinks(
	ctx context.Context, from string, targets []string,
) (map[string]domain.ResolvedLink, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.Targets.Execute(ctx, v.ID, from, targets)
}
