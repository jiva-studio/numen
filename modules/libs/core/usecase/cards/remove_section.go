package cards

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// RemoveSection takes the heading of the section standing at a place, counted from the first, away. No card is removed.
type RemoveSection struct {
	read  Read
	write Write
}

func NewRemoveSection(read Read, write Write) RemoveSection {
	return RemoveSection{read: read, write: write}
}

func (u RemoveSection) Execute(ctx context.Context, v domain.Vault, at DeckVersion, section int) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, at, func(_ DeckContents, file *format.DeckFile) error {
		return file.RemoveSection(section)
	})
	return changed.result, err
}
