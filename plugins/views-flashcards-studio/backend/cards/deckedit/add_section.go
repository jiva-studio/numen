package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// AddSection writes a section at the end of a deck.
type AddSection struct {
	read  Read
	write Write
}

// NewAddSection creates the AddSection scenario.
func NewAddSection(read Read, write Write) AddSection { return AddSection{read: read, write: write} }

// Execute adds a section to a deck.
func (u AddSection) Execute(ctx context.Context, v domain.Vault, version DeckVersion, name string) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(_ DeckContents, file *format.DeckFile) error {
		return file.AddSection(format.Section{Name: name, Preamble: ""})
	})
	return changed.result, err
}
