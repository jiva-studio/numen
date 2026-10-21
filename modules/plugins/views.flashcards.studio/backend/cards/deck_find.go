package cards

import (
	"fmt"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
)

func findCard(held []format.Card, mark string) (int, error) {
	at := -1
	if mark != "" {
		for i, card := range held {
			if string(card.Mark) != mark {
				continue
			}
			if at >= 0 {
				return 0, fmt.Errorf("%w: %s", format.ErrTwoCards, mark)
			}
			at = i
		}
	}
	if at < 0 {
		return 0, fmt.Errorf("%w: %s", format.ErrNoSuchCard, mark)
	}
	return at, nil
}

func getMark(given []format.CardMark, at int) string {
	for _, one := range given {
		if one.Card == at {
			return string(one.Mark)
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
