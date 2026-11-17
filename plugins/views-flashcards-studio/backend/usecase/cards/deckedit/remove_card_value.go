package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// RemoveCardValue takes one field off the card of a card id.
type RemoveCardValue struct {
	read  ports.DeckReader
	write ports.DeckWriter
}

func NewRemoveCardValue(read ports.DeckReader, write ports.DeckWriter) RemoveCardValue {
	return RemoveCardValue{read: read, write: write}
}

func (u RemoveCardValue) Execute(
	ctx context.Context, v domain.Vault, version DeckVersion, cardID, field string,
) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(_ format.DeckContents, file *format.DeckFile) error {
		return file.RemoveValue(format.ID(cardID), field)
	})
	return changed.result, err
}
