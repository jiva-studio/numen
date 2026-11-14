package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// RenameSection gives the section standing at a place, counted from the first, a different name.
type RenameSection struct {
	read  ports.DeckReader
	write ports.DeckWriter
}

func NewRenameSection(read ports.DeckReader, write ports.DeckWriter) RenameSection {
	return RenameSection{read: read, write: write}
}

func (u RenameSection) Execute(
	ctx context.Context, v domain.Vault, version DeckVersion, section int, name string,
) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(_ format.DeckContents, file *format.DeckFile) error {
		return file.RenameSection(section, name)
	})
	return changed.result, err
}
