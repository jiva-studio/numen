package deckedit

import (
	"fmt"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

func findCard(target []format.Card, mark string) (int, error) {
	pos := -1
	if mark != "" {
		for i, card := range target {
			if string(card.Mark) != mark {
				continue
			}
			if pos >= 0 {
				return 0, fmt.Errorf("%w: %s", format.ErrTwoCards, mark)
			}
			pos = i
		}
	}
	if pos < 0 {
		return 0, fmt.Errorf("%w: %s", format.ErrNoSuchCard, mark)
	}
	return pos, nil
}

func getMark(marks []format.CardMark, pos int) string {
	for _, one := range marks {
		if one.Card == pos {
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
