package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// RemoveCardValue takes one field off the card of a card id.
type RemoveCardValue struct {
	read  Read
	write Write
}

// NewRemoveCardValue creates the RemoveCardValue scenario.
func NewRemoveCardValue(read Read, write Write) RemoveCardValue {
	return RemoveCardValue{read: read, write: write}
}

// Execute removes one value from a card.
func (u RemoveCardValue) Execute(
	ctx context.Context, v domain.Vault, version DeckVersion, cardID, field string,
) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(_ DeckContents, file *format.DeckFile) error {
		return file.RemoveValue(cardid.ID(cardID), field)
	})
	return changed.result, err
}
