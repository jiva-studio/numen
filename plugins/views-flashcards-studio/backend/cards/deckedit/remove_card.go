package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// RemoveCard takes the card of a mark out of a deck.
type RemoveCard struct {
	read  Read
	write Write
}

func NewRemoveCard(read Read, write Write) RemoveCard { return RemoveCard{read: read, write: write} }

func (u RemoveCard) Execute(ctx context.Context, v domain.Vault, version DeckVersion, mark string) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(_ DeckContents, file *format.DeckFile) error {
		return file.RemoveCard(cardid.ID(mark))
	})
	return changed.result, err
}
