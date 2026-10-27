package cards

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// CardModelSummary is one card model as somebody choosing between them sees it:
// where the file is, what it is called, and what a card cut by it is asked for.
type CardModelSummary struct {
	Path   string
	Title  string
	Fields []string
}

// List is every card model a vault holds.
//
// The index says which notes are card models and the files say what each declares,
// so a card model that cannot be read is on the list with no fields on it.
type List struct {
	readers ports.VaultReaders
	notes   CardModelQueries
}

// CardModelQueries is the one question listing them asks of the index.
type CardModelQueries interface {
	// Card models is every card model one vault holds, by path. A card names the
	// card model it is cut by, and this is the list those names are picked from.
	GetCardModels(ctx context.Context, vaultID domain.VaultID) ([]ports.CardModel, error)
}

// NewList is what the vault's card models are listed through: what says which
// notes are card models, and the vault each is read out of for its fields.
func NewList(readers ports.VaultReaders, notes CardModelQueries) List {
	return List{readers: readers, notes: notes}
}

// Execute lists the card models of one vault, by path, and says how many the vault
// holds.
//
// Limit is how many of them are read, and zero or less is all of them. The
// index says how many there are, so the count stands above the list without a
// file being opened for the card models left out.
func (u List) Execute(
	ctx context.Context, v domain.Vault, limit int,
) ([]CardModelSummary, int, error) {
	all, err := u.notes.GetCardModels(ctx, v.ID)
	if err != nil {
		return nil, 0, err
	}
	target := all
	if limit > 0 {
		target = all[:min(limit, len(all))]
	}

	// A card model holds no cards, so nothing here asks where a wikilink lands.
	read := NewRead(u.readers, nil)
	out := make([]CardModelSummary, 0, len(target))
	for _, s := range target {
		items := CardModelSummary{Path: s.Path, Title: s.Title}
		cardModel, err := read.CardModel(ctx, v, s.Path)
		if err != nil {
			return nil, 0, err
		}
		if cardModel.Outcome == note.Ok {
			items.Fields = cardModel.Body.Fields
		}
		out = append(out, items)
	}
	return out, len(all), nil
}
