package cards

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
)

// GetCardPage reads a run of the cards of one deck, or one card of it.
type GetCardPage struct{ read Read }

func NewGetCardPage(read Read) GetCardPage { return GetCardPage{read: read} }

func (u GetCardPage) Execute(ctx context.Context, v domain.Vault, in Query) (Page, error) {
	contents, err := readExistingDeck(ctx, u.read, v, in.Path)
	if err != nil {
		return Page{}, err
	}
	res := Page{
		Total:       len(contents.Body.Cards),
		Problems:    contents.Body.Problems,
		Fingerprint: contents.Fingerprint,
	}
	for _, s := range contents.Body.Sections {
		res.Sections = append(res.Sections, s.Name)
	}
	if in.Mark != "" {
		pos, err := findCard(contents.Body.Cards, in.Mark)
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
