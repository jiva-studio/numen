package cards

import "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"

// Whole is a deck's body with every card made whole: a card carrying no mark is
// given one, and a heading that has fallen out of step with the card's first
// field is written again from the field. Every other byte of the body is left
// as it arrived.
//
// Neither rule can be kept where a deck's body is composed — one needs a
// generator, the other needs the card's stencil — so a deck passes here once, on
// its way to the vault.
//
// Stencils is the stencil each card is cut by, keyed by what stands in the
// card's brackets. A card whose stencil is not among them, or whose stencil
// declares no field, is not reprojected: nothing can say which of its fields is
// first, so its heading is left exactly as it stands. It is given its mark all
// the same, because a mark is what the card is and no stencil says so.
//
// The body is text whose line endings are normalised, and so is what comes
// back. The endings the file keeps are put on when it is written.
func Whole(body string, stencils map[string]Stencil, mint func() (string, error)) (string, error) {
	raw := []byte(body)
	deck, spans := readDeck(domain.FileRef{}, raw)

	// Backwards, because a splice moves every byte after it.
	for i := len(spans) - 1; i >= 0; i-- {
		card, span := deck.Cards[i], spans[i]

		carried := card.Mark
		if carried == "" {
			minted, err := mint()
			if err != nil {
				return "", err
			}
			carried = minted
		}
		text := card.Heading
		if first := stencils[card.Stencil].First(); first != "" {
			value, _ := card.Value(first)
			text = Project(value)
		}

		heading := WriteHeading(text, carried)
		if heading == span.name {
			continue
		}
		line := headingLine(2, heading)
		if span.from < len(raw) {
			line += "\n"
		}
		out := make([]byte, 0, len(raw)-(span.from-span.head)+len(line))
		out = append(out, raw[:span.head]...)
		out = append(out, line...)
		raw = append(out, raw[span.from:]...)
	}
	return string(raw), nil
}
