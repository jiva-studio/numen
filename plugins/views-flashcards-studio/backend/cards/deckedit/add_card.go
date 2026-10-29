// Package deckedit holds the scenarios that edit the cards and sections of a deck.
package deckedit

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// AddCard writes a card at the end of a deck, or at the end of one of its sections.
type AddCard struct {
	read  Read
	write Write
}

// NewAddCard creates the AddCard scenario.
func NewAddCard(read Read, write Write) AddCard { return AddCard{read: read, write: write} }

// Execute adds a card to a deck or one of its sections.
func (u AddCard) Execute(
	ctx context.Context, v domain.Vault, version DeckVersion, cardModel string, values []format.Value, section *int,
) (AddedCard, error) {
	position := 0
	changed, err := changeDeck(ctx, u.read, u.write, v, version, func(contents DeckContents, file *format.DeckFile) error {
		card := format.Card{Heading: "", CardID: "", CardModelLink: cardModel, Preamble: "", Section: 0, Values: values}
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
	return AddedCard{EditResult: changed.result, CardID: getCardID(changed.cardIDs, position)}, nil
}
