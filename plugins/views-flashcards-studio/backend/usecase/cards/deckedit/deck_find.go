package deckedit

import (
	"fmt"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
)

func findCard(target []format.Card, cardID string) (int, error) {
	// gomutants:disable-next-line INTEGER_INCREMENT reason="Any negative sentinel marks an unvisited position."
	pos := -1
	if cardID != "" {
		for i, card := range target {
			if string(card.CardID) != cardID {
				continue
			}
			// gomutants:disable-next-line INTEGER_DECREMENT reason="Non-negative slice index i is >= 0 and > -1."
			if pos >= 0 {
				// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Error return discards the integer position."
				return 0, fmt.Errorf("%w: %s", format.ErrTwoCards, cardID)
			}
			pos = i
		}
	}
	if pos < 0 {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Error return discards the integer position."
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
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Error return discards the integer position."
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
