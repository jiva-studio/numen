package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cardid"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
)

// EditCard writes values into the card of a mark, and cuts it by stencil from then on when that is not empty.
type EditCard struct {
	read  Read
	write Write
}

func NewEditCard(read Read, write Write) EditCard { return EditCard{read: read, write: write} }

func (u EditCard) Execute(
	ctx context.Context, v domain.Vault, version DeckVersion, mark, stencil string, values []format.Value,
) (EditResult, error) {
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(contents DeckContents, file *format.DeckFile) error {
		card := cardid.ID(mark)
		if _, err := findCard(contents.Body.Cards, mark); err != nil {
			return err
		}
		if stencil != "" {
			if err := file.SetStencil(card, stencil); err != nil {
				return err
			}
		}
		for _, one := range values {
			if err := file.SetValue(card, one.Field, one.Text); err != nil {
				return err
			}
		}
		return nil
	})
	return changed.result, err
}
