package cards

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
)

// RenameSection gives the section standing at a place, counted from the first, a different name.
type RenameSection struct {
	read  Read
	write Write
}

func NewRenameSection(read Read, write Write) RenameSection {
	return RenameSection{read: read, write: write}
}

func (u RenameSection) Execute(
	ctx context.Context, v domain.Vault, version DeckVersion, section int, name string,
) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(_ DeckContents, file *format.DeckFile) error {
		return file.RenameSection(section, name)
	})
	return changed.result, err
}
