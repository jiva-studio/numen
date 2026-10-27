package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// RemoveCardValue takes one field off the card of a card id.
type RemoveCardValue struct {
	read  Read
	write Write
}

func NewRemoveCardValue(read Read, write Write) RemoveCardValue {
	return RemoveCardValue{read: read, write: write}
}

func (u RemoveCardValue) Execute(
	ctx context.Context, v domain.Vault, version DeckVersion, cardID, field string,
) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(_ DeckContents, file *format.DeckFile) error {
		return file.RemoveValue(cardid.ID(cardID), field)
	})
	return changed.result, err
}
