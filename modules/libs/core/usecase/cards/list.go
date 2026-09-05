package cards

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// StencilSummary is one stencil as somebody choosing between them sees it:
// where the file is, what it is called, and what a card cut by it is asked for.
type StencilSummary struct {
	Path   string
	Title  string
	Fields []string
}

// List is every stencil a vault holds.
//
// The index says which notes are stencils and the files say what each declares,
// so a stencil that cannot be read is on the list with no fields on it.
type List struct {
	Readers port.VaultReaders
	Notes   StencilQueries
}

// StencilQueries is the one question listing them asks of the index.
type StencilQueries interface {
	// Stencils is every stencil one vault holds, by path. A card names the
	// stencil it is cut by, and this is the list those names are picked from.
	Stencils(ctx context.Context, vaultID domain.VaultID) ([]domain.Stencil, error)
}

// NewList is what the vault's stencils are listed through: what says which
// notes are stencils, and the vault each is read out of for its fields.
func NewList(readers port.VaultReaders, notes StencilQueries) List {
	return List{Readers: readers, Notes: notes}
}

// Execute lists the stencils of one vault, by path, and says how many the vault
// holds.
//
// Limit is how many of them are read, and zero or less is all of them. The
// index says how many there are, so the count stands above the list without a
// file being opened for the stencils left out.
func (u List) Execute(
	ctx context.Context, v domain.Vault, limit int,
) ([]StencilSummary, int, error) {
	all, err := u.Notes.Stencils(ctx, v.ID)
	if err != nil {
		return nil, 0, err
	}
	held := all
	if limit > 0 && limit < len(held) {
		held = held[:limit]
	}

	// A stencil holds no cards, so nothing here asks where a wikilink lands.
	read := Read{Readers: u.Readers}
	out := make([]StencilSummary, 0, len(held))
	for _, s := range held {
		listed := StencilSummary{Path: s.Path, Title: s.Title}
		stencil, err := read.Stencil(ctx, v, s.Path)
		if err != nil {
			return nil, 0, err
		}
		if stencil.Outcome == note.Ok {
			listed.Fields = stencil.Body.Fields
		}
		out = append(out, listed)
	}
	return out, len(all), nil
}
