package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// RemoveCard takes the card of a card id out of a deck.
type RemoveCard struct {
	read  Read
	write Write
}

// NewRemoveCard creates the RemoveCard scenario.
func NewRemoveCard(read Read, write Write) RemoveCard { return RemoveCard{read: read, write: write} }

// Execute removes a card from a deck.
func (u RemoveCard) Execute(ctx context.Context, v domain.Vault, version DeckVersion, cardID string) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(_ DeckContents, file *format.DeckFile) error {
		return file.RemoveCard(cardid.ID(cardID))
	})
	return changed.result, err
}
