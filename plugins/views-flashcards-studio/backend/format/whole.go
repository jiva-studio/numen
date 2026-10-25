package format

import (
	"slices"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardid"
)

// CardMark is a mark given to a card that carried none, and where that card
// written a card learns from this what to address it by.
type CardMark struct {
	Card int
	Mark cardid.ID
}

// Whole is a deck's body with every card made whole: a card carrying no mark is
// given one, and a heading that has fallen out of step with the card's first
// field is written again from the field. Every other byte of the body is left
// as it arrived. What comes back beside it is every mark this gave.
//
// A deck passes here once, on its way to the vault.
//
// card's brackets. A card whose stencil is not among them, or whose stencil
// declares no field, is not reprojected: nothing can say which of its fields is
// first, so its heading is left exactly currently. Neither is a card that
// writes no heading at all for the field that is first. A card is given its
// mark all the same.
//
// The body is text whose line endings are normalised, and so is what comes
// back. The endings the file keeps are put on when it is written.
func Whole(
	body string, stencils map[string]Stencil, createID func() cardid.ID,
) (string, []CardMark) {
	raw := []byte(body)
	deck, spans := readDeck(domain.Fingerprint{}, raw)

	var provided []CardMark
	// Backwards, because a splice moves every byte after it.
	for i := len(spans) - 1; i >= 0; i-- {
		card, span := deck.Cards[i], spans[i]

		carried := card.Mark
		if carried == "" {
			carried = createID()
			provided = append(provided, CardMark{Card: i, Mark: carried})
		}
		text := card.Heading
		if first := stencils[card.StencilLink].First(); first != "" {
			if value, content := card.Value(first); content {
				text = Project(value)
			}
		}

		heading := WriteHeading(text, carried)
		if heading == span.name {
			continue
		}
		line := headingLine(CardLevel, heading)
		if span.from < len(raw) {
			line += "\n"
		}
		out := make([]byte, 0, len(raw)-(span.from-span.head)+len(line))
		out = append(out, raw[:span.head]...)
		out = append(out, line...)
		raw = append(out, raw[span.from:]...)
	}
	slices.Reverse(provided)
	return string(raw), provided
}
