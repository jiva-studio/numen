package deckedit

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// GetCardPage reads a run of the cards of one deck, or one card of it.
type GetCardPage struct{ read Read }

// NewGetCardPage creates the GetCardPage scenario.
func NewGetCardPage(read Read) GetCardPage { return GetCardPage{read: read} }

// Execute reads one page of the cards of a deck.
func (u GetCardPage) Execute(ctx context.Context, v domain.Vault, in Query) (Page, error) {
	contents, err := readExistingDeck(ctx, u.read, v, in.Path)
	if err != nil {
		return Page{}, err
	}
	res := Page{
		Cards:    nil,
		Sections: nil,
		Total:    len(contents.Body.Cards),
		Problems: contents.Body.Problems,
		FileStat: contents.FileStat,
	}
	for _, s := range contents.Body.Sections {
		res.Sections = append(res.Sections, s.Name)
	}
	if in.CardID != "" {
		pos, err := findCard(contents.Body.Cards, in.CardID)
		if err != nil {
			return Page{}, err
		}
		res.Cards = append(res.Cards, keepFields(contents.Body.Cards[pos], in.Fields))
		return res, nil
	}
	for _, card := range getCardRange(contents.Body.Cards, in.From, in.Limit) {
		res.Cards = append(res.Cards, keepFields(card, in.Fields))
	}
	return res, nil
}

func getCardRange(deck []format.Card, from, limit int) []format.Card {
	from = min(max(from, 0), len(deck))
	return deck[from:min(from+max(limit, 0), len(deck))]
}

func keepFields(card format.Card, fields []string) format.Card {
	if len(fields) == 0 {
		return card
	}
	filteredCard := card
	filteredCard.Values = nil
	for _, v := range card.Values {
		if slices.Contains(fields, v.Field) {
			filteredCard.Values = append(filteredCard.Values, v)
		}
	}
	return filteredCard
}
