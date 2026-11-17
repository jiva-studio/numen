package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// RemoveCard takes the card of a card id out of a deck.
type RemoveCard struct {
	read  ports.DeckReader
	write ports.DeckWriter
}

func NewRemoveCard(read ports.DeckReader, write ports.DeckWriter) RemoveCard {
	return RemoveCard{read: read, write: write}
}

func (u RemoveCard) Execute(ctx context.Context, v domain.Vault, version DeckVersion, cardID string) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(_ format.DeckContents, file *format.DeckFile) error {
		return file.RemoveCard(format.ID(cardID))
	})
	return changed.result, err
}
