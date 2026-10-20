package cards

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// AddSection writes a section at the end of a deck.
type AddSection struct {
	read  Read
	write Write
}

func NewAddSection(read Read, write Write) AddSection { return AddSection{read: read, write: write} }

func (u AddSection) Execute(ctx context.Context, v domain.Vault, at DeckVersion, name string) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, at, func(_ DeckContents, file *format.DeckFile) error {
		return file.AddSection(format.Section{Name: name})
	})
	return changed.result, err
}
