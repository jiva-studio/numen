package format

import (
	"slices"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/cardid"
)

// CardID is a card id given to a card that carried none, and where that card
// written a card learns from this what to address it by.
type CardID struct {
	Card   int
	CardID cardid.ID
}

// Whole is a deck's body with every card made whole: a card carrying no card id is given one,
// and a heading out of step with the card's first field is written again from the field. Every
// other byte is left as it arrived, and the card ids given are returned with it. A card whose
// card model is not in cardModels, or declares no field, or writes no heading for its first
// field keeps its heading, but is still given a card id. The body has normalized line endings.
func Whole(
	body string, cardModels map[string]CardModel, createID func() cardid.ID,
) (string, []CardID) {
	raw := []byte(body)
	deck, spans := readDeck(createEmptyStat(), raw)

	var provided []CardID
	// Backwards, because a splice moves every byte after it.
	for i, span := range slices.Backward(spans) {
		card := deck.Cards[i]

		carried := card.CardID
		if carried == "" {
			carried = createID()
			provided = append(provided, CardID{Card: i, CardID: carried})
		}
		text := card.Heading
		if first := cardModels[card.CardModelLink].First(); first != "" {
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
		// gomutants:disable-next-line ARITHMETIC_BASE,INVERT_NEGATIVES reason="Slice capacity hint does not affect observable output."
		out := make([]byte, 0, len(raw)-(span.from-span.head)+len(line))
		out = append(out, raw[:span.head]...)
		out = append(out, line...)
		raw = append(out, raw[span.from:]...)
	}
	slices.Reverse(provided)
	return string(raw), provided
}
