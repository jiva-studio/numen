package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
)

// AddCard writes a card at the end of a deck, or at the end of one of its sections.
type AddCard struct {
	read  Read
	write Write
}

func NewAddCard(read Read, write Write) AddCard { return AddCard{read: read, write: write} }

func (u AddCard) Execute(
	ctx context.Context, v domain.Vault, version DeckVersion, stencil string, values []format.Value, section *int,
) (AddedCard, error) {
	position := 0
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(contents DeckContents, file *format.DeckFile) error {
		card := format.Card{StencilLink: stencil, Values: values}
		if section == nil {
			position = len(contents.Body.Cards)
			return file.AddCard(card)
		}
		end, err := findSectionEnd(contents.Body, *section)
		if err != nil {
			return err
		}
		position = end
		return file.AddCardUnder(*section, card)
	})
	if err != nil {
		return AddedCard{}, err
	}
	return AddedCard{EditResult: changed.result, Mark: getMark(changed.marks, position)}, nil
}
