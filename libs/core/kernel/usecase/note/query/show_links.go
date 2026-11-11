package query

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// VaultFinding resolves what the user typed, a name, a path or an identity, into a vault on the list.
type VaultFinding func(nameOrPath string) (domain.Vault, error)

// NoteLinks is what one note is joined to: what it points at, and what points
// at it. Both halves together, because looking at one without the other is how
// a graph gets misread — a note with no outgoing links may still be the center
// of the vault.
type NoteLinks struct {
	Vault     domain.Vault
	Links     []domain.ResolvedLink
	Backlinks []domain.ResolvedLink
}

// ShowLinks gathers them for one note.
type ShowLinks struct {
	find  VaultFinding
	links noteport.LinkQueries
}

// NewShowLinks returns the use case built on both halves: the index that returns what a
// note points at and what points at it.
func NewShowLinks(links noteport.LinkQueries) ShowLinks {
	return ShowLinks{links: links}
}

// WithFind sets the lookup that resolves a name, path or identity into a vault.
func (u ShowLinks) WithFind(find VaultFinding) ShowLinks {
	u.find = find
	return u
}

// ExecuteByName resolves the vault and gathers the links for one note.
func (u ShowLinks) ExecuteByName(ctx context.Context, nameOrPath, path string) (NoteLinks, error) {
	v, err := u.find(nameOrPath)
	if err != nil {
		return NoteLinks{}, err
	}
	return u.Execute(ctx, v, path)
}

func (u ShowLinks) Execute(ctx context.Context, v domain.Vault, path string) (NoteLinks, error) {
	links, err := u.links.GetLinks(ctx, v.ID, path)
	if err != nil {
		return NoteLinks{}, err
	}
	backlinks, err := u.links.GetBacklinks(ctx, v.ID, path)
	if err != nil {
		return NoteLinks{}, err
	}
	return NoteLinks{Vault: v, Links: links, Backlinks: backlinks}, nil
}
