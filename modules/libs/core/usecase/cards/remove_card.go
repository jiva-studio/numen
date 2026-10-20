package cards

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// RemoveCard takes the card of a mark out of a deck.
type RemoveCard struct {
	read  Read
	write Write
}

func NewRemoveCard(read Read, write Write) RemoveCard { return RemoveCard{read: read, write: write} }

func (u RemoveCard) Execute(ctx context.Context, v domain.Vault, at DeckVersion, mark string) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, at, func(_ DeckContents, file *format.DeckFile) error {
		return file.RemoveCard(domain.CardID(mark))
	})
	return changed.result, err
}
