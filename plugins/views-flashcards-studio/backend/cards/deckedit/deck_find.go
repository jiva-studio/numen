package deckedit

import (
	"fmt"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

func findCard(target []format.Card, cardID string) (int, error) {
	pos := -1
	if cardID != "" {
		for i, card := range target {
			if string(card.CardID) != cardID {
				continue
			}
			if pos >= 0 {
				return 0, fmt.Errorf("%w: %s", format.ErrTwoCards, cardID)
			}
			pos = i
		}
	}
	if pos < 0 {
		return 0, fmt.Errorf("%w: %s", format.ErrNoSuchCard, cardID)
	}
	return pos, nil
}

func getCardID(cardIDs []format.CardID, pos int) string {
	for _, one := range cardIDs {
		if one.Card == pos {
			return string(one.CardID)
		}
	}
	return ""
}

func findSectionEnd(d format.Deck, section int) (int, error) {
	if section < 0 || section >= len(d.Sections) {
		return 0, fmt.Errorf(
			"this deck has %d sections, so there is none standing at %d", len(d.Sections), section)
	}
	for i, card := range d.Cards {
		if card.Section > section {
			return i, nil
		}
	}
	return len(d.Cards), nil
}
