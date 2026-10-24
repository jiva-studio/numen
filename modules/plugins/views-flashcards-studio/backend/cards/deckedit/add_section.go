package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/format"
)

// AddSection writes a section at the end of a deck.
type AddSection struct {
	read  Read
	write Write
}

func NewAddSection(read Read, write Write) AddSection { return AddSection{read: read, write: write} }

func (u AddSection) Execute(ctx context.Context, v domain.Vault, version DeckVersion, name string) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(_ DeckContents, file *format.DeckFile) error {
		return file.AddSection(format.Section{Name: name})
	})
	return changed.result, err
}
