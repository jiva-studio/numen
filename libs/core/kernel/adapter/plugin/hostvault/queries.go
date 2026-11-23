package hostvault

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

// Lookups reads the index of the vault: the data of its notes and the links between them.
type Lookups struct {
	Notes   Notes
	Changes Changes
	Links   Links
}

// Notes reads the index data of the notes of the vault.
type Notes struct {
	Notes      query.GetNotes
	Types      *query.GetTypes
	NamedPaths *query.ListNamedPaths
	TypedPaths query.ListPathsOfType

	session sessionAccess
}

// Changes reads the frontmatter of the notes of the vault and its file change feed.
type Changes struct {
	Frontmatter query.ReadFrontmatter
	FileChanges query.ListFileChanges

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

func (n Notes) ListNamedPaths(ctx context.Context, name string) ([]string, error) {
	v, err := n.session.open()
	if err != nil {
		return nil, err
	}
	return n.NamedPaths.Execute(ctx, v, name)
}

func (n Notes) ListPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	v, err := n.session.open()
	if err != nil {
		return nil, err
	}
	return n.TypedPaths.Execute(ctx, v, noteType)
}

// Links reads the links between the notes of the vault.
type Links struct {
	Links     query.ListLinks
	Backlinks query.ListBacklinks
	Targets   *query.ResolveLinks

	session sessionAccess
}

// ListLinks returns the links of the note at the path.
func (l Links) ListLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.Links.Execute(ctx, v, from)
}

// ListBacklinks returns the links to the note at the path.
func (l Links) ListBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error) {
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

func (n Changes) ReadFrontmatter(ctx context.Context, q frontmatter.Query) (frontmatter.Page, error) {
	v, err := n.session.open()
	if err != nil {
		return frontmatter.Page{}, err
	}
	return n.Frontmatter.Execute(ctx, v, q)
}

func (n Changes) ListFileChanges(ctx context.Context, q filechange.Query) (filechange.Page, error) {
	v, err := n.session.open()
	if err != nil {
		return filechange.Page{}, err
	}
	return n.FileChanges.Execute(ctx, v, q)
}
