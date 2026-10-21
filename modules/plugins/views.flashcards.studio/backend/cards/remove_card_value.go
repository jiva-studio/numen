package cards

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cardid"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
)

// RemoveCardValue takes one field off the card of a mark.
type RemoveCardValue struct {
	read  Read
	write Write
}

func NewRemoveCardValue(read Read, write Write) RemoveCardValue {
	return RemoveCardValue{read: read, write: write}
}

func (u RemoveCardValue) Execute(
	ctx context.Context, v domain.Vault, at DeckVersion, mark, field string,
) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, at, func(_ DeckContents, file *format.DeckFile) error {
		return file.RemoveValue(cardid.ID(mark), field)
	})
	return changed.result, err
}
