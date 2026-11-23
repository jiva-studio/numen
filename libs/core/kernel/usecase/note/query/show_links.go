package query

import (
	"context"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

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
	links noteport.LinkQueries
}

// NewShowLinks returns the use case built on both halves: the index that returns what a
// note points at and what points at it.
func NewShowLinks(links noteport.LinkQueries) ShowLinks {
	return ShowLinks{links: links}
}

// ShowLinksByName gathers the links of one note of the vault a person names.
type ShowLinksByName struct {
	lookup vaultport.VaultLookup
	show   ShowLinks
}

// NewShowLinksByName creates the use case that resolves the vault with find and gathers the links with show.
func NewShowLinksByName(lookup vaultport.VaultLookup, show ShowLinks) ShowLinksByName {
	return ShowLinksByName{lookup: lookup, show: show}
}

// ExecuteByName resolves the vault and gathers the links for one note.
func (u ShowLinksByName) ExecuteByName(ctx context.Context, nameOrPath, path string) (NoteLinks, error) {
	v, err := vaultdomain.FindByName(nameOrPath, u.lookup.FindVault)
	if err != nil {
		return NoteLinks{}, err
	}
	return u.show.Execute(ctx, v, path)
}

func (u ShowLinks) Execute(ctx context.Context, v domain.Vault, path string) (NoteLinks, error) {
	links, err := u.links.ListLinks(ctx, v.ID, path)
	if err != nil {
		return NoteLinks{}, err
	}
	backlinks, err := u.links.ListBacklinks(ctx, v.ID, path)
	if err != nil {
		return NoteLinks{}, err
	}
	return NoteLinks{Vault: v, Links: links, Backlinks: backlinks}, nil
}
