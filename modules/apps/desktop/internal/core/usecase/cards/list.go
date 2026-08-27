package cards

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// Listed is one stencil as somebody choosing between them sees it: where the
// file is, what it is called, and what a card cut by it is asked for.
type Listed struct {
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
	Notes   port.NoteQueries
}

// Execute lists the stencils of one vault, by path.
func (u List) Execute(ctx context.Context, v domain.Vault) ([]Listed, error) {
	held, err := u.Notes.Stencils(ctx, v.ID)
	if err != nil {
		return nil, err
	}

	read := Read{Readers: u.Readers}
	out := make([]Listed, 0, len(held))
	for _, s := range held {
		listed := Listed{Path: s.Path, Title: s.Title}
		stencil, err := read.Stencil(ctx, v, s.Path)
		if err != nil {
			return nil, err
		}
		if stencil.Outcome == note.Ok {
			listed.Fields = stencil.Stencil.Fields
		}
		out = append(out, listed)
	}
	return out, nil
}
