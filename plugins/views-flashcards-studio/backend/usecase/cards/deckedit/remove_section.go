package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// RemoveSection takes the heading of the section standing at a place, counted from the first, away. No card is removed.
type RemoveSection struct {
	read  ports.DeckReader
	write ports.DeckWriter
}

func NewRemoveSection(read ports.DeckReader, write ports.DeckWriter) RemoveSection {
	return RemoveSection{read: read, write: write}
}

func (u RemoveSection) Execute(ctx context.Context, v domain.Vault, version DeckVersion, section int) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(_ format.DeckContents, file *format.DeckFile) error {
		return file.RemoveSection(section)
	})
	return changed.result, err
}
