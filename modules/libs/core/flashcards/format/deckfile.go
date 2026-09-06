package format

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// ErrNoSuchCard is what changing a card says when the deck holds no card of
// that mark. A card is known by its mark, so a card rewritten from end to end
// is still that card.
var ErrNoSuchCard = errors.New("this deck holds no card of that mark")

// ErrTwoCards is what changing a card says when two cards of the deck carry
// the mark it was addressed by. Both are read and both are shown, and which of
// the two is meant is a thing only the person who wrote them knows.
var ErrTwoCards = errors.New("two cards of this deck carry that mark")

// ErrNoSuchSection is what changing a section says when the deck holds no
// section standing there.
var ErrNoSuchSection = errors.New("this deck holds no section standing there")

// ErrNoSuchValue is what removing a value says when the card writes nothing
// under that field. The stencil may well declare it: what is not there is what
// the person wrote.
var ErrNoSuchValue = errors.New("this card writes no value under that field")

// DeckFile is a deck held open so that one value can be changed and every other
// byte of the file left as it arrived. A change is a splice: the run one value
// occupies is replaced, and nothing else in the file is written.
type DeckFile struct {
	doc *markdown.Document
}

// OpenDeck reads a deck for changing.
func OpenDeck(raw []byte) (*DeckFile, error) {
	doc, err := markdown.Open(raw)
	if err != nil {
		return nil, err
	}
	return &DeckFile{doc: doc}, nil
}

// OpenDeckBody holds a deck's body open for changing, with no frontmatter
// around it. A caller that reads a note and writes its body back changes cards
// through this, and every byte no change reached is the byte it arrived as.
func OpenDeckBody(body string) *DeckFile {
	return &DeckFile{doc: markdown.OpenBody([]byte(body))}
}

// Bytes is the deck as it now stands.
func (f *DeckFile) Bytes() []byte { return f.doc.Bytes() }

// Body is the deck's prose as it now stands.
func (f *DeckFile) Body() string { return f.doc.Body() }

// Stamped writes the identifier a file carrying none is to carry, and reports
// whether it wrote one. The application changing what a note holds is what
// writes an identifier into it.
func (f *DeckFile) Stamped(identifier string) (bool, error) { return stamped(f.doc, identifier) }

// Deck is what the file now says.
func (f *DeckFile) Deck(ref domain.Fingerprint) Deck {
	deck, _ := readDeck(ref, []byte(f.doc.Body()))
	return deck
}

// Whole makes every card of the file whole and reports the marks it minted,
// which is what Whole does to a body. The file keeps its own line endings.
func (f *DeckFile) Whole(
	stencils map[string]Stencil, mint func() (domain.CardID, error),
) ([]MintedMark, error) {
	body, minted, err := Whole(markdown.Normalised(f.doc.Body()), stencils, mint)
	if err != nil {
		return nil, err
	}
	if err := f.doc.SetBody(body); err != nil {
		return nil, err
	}
	return minted, nil
}

// SetValue writes what the card of a mark holds under one field. A field the
// card does not carry yet is added at the end of it, which leaves the fields it
// does carry in the order the person wrote them.
//
// A mark two cards of the deck carry addresses neither: choosing between them
// is ErrTwoCards.
//
// Writing the first field is what rewrites the heading, and that is done where
// the deck is made whole, not here.
func (f *DeckFile) SetValue(card domain.CardID, field, value string) error {
	body := []byte(f.doc.Body())
	span, err := f.carrying(body, card)
	if err != nil {
		return err
	}
	for _, v := range span.values {
		if v.field == field {
			return f.doc.SpliceBody(v.from, v.to, under(value, v.to == len(body)))
		}
	}
	block := headingLine(FieldLevel, oneLine(field))
	if text := trimBlankLines(markdown.Normalised(value)); text != "" {
		block += "\n\n" + text
	}
	return f.doc.SpliceBody(span.end, span.end, insert(body, span.end, block))
}

// RemoveValue takes one field off a card: its heading line and what stands
// under it. The fields around it keep the order the person wrote them in, and
// a card writing nothing under that field is ErrNoSuchValue.
//
// Taking the first field off is what rewrites the heading, and that is done
// where the deck is made whole, not here.
func (f *DeckFile) RemoveValue(card domain.CardID, field string) error {
	body := []byte(f.doc.Body())
	span, err := f.carrying(body, card)
	if err != nil {
		return err
	}
	for _, v := range span.values {
		if v.field != field {
			continue
		}
		// The blank line under the value went with it, so what follows closes
		// up against what the value stood beneath.
		head, to := v.head, v.to
		written := ""
		// A value nothing stood under was last in the file, and what is above
		// it now ends on the one break a file ends with.
		if to == len(body) {
			head = trimmedEnd(body, 0, head)
			if head > 0 {
				written = "\n"
			}
		}
		return f.doc.SpliceBody(head, to, written)
	}
	return fmt.Errorf("%w: %s", ErrNoSuchValue, field)
}

// SetStencil writes the wikilink naming the stencil a card is cut by. A card
// that named none is given the paragraph under its heading, and the values it
// already holds stay where they stand.
func (f *DeckFile) SetStencil(card domain.CardID, name string) error {
	body := []byte(f.doc.Body())
	span, err := f.carrying(body, card)
	if err != nil {
		return err
	}
	link := "[[" + oneLine(name) + "]]"
	if span.linkTo > span.linkFrom {
		return f.doc.SpliceBody(span.linkFrom, span.linkTo, link)
	}
	return f.doc.SpliceBody(span.from, span.from, "\n"+link+"\n")
}

// RemoveCard takes a card out of the deck: its heading, the stencil it named
// and every value under it. What stood above and below is left where it was.
func (f *DeckFile) RemoveCard(card domain.CardID) error {
	body := []byte(f.doc.Body())
	span, err := f.carrying(body, card)
	if err != nil {
		return err
	}
	// The blank line under the card went with it, so what follows closes up
	// against what the card stood beneath.
	head, to := span.head, span.end
	written := ""
	// A card nothing stood under was last, and what is above it now ends on the
	// one break a file ends with.
	if to == len(body) {
		head = trimmedEnd(body, 0, head)
		if head > 0 {
			written = "\n"
		}
	}
	return f.doc.SpliceBody(head, to, written)
}

// carrying is where the card of a mark stands. A deck holding none of that mark
// is ErrNoSuchCard, and a deck holding two is ErrTwoCards: choosing between
// them is choosing which of the two the person meant.
func (f *DeckFile) carrying(body []byte, card domain.CardID) (cardSpan, error) {
	_, spans := readDeck(domain.Fingerprint{}, body)
	at := -1
	for i, span := range spans {
		if span.mark == "" || span.mark != card {
			continue
		}
		if at >= 0 {
			return cardSpan{}, fmt.Errorf("%w: %s", ErrTwoCards, card)
		}
		at = i
	}
	if at < 0 {
		return cardSpan{}, fmt.Errorf("%w: %s", ErrNoSuchCard, card)
	}
	return spans[at], nil
}

// RenameSection gives the section standing at one place in the deck another
// name. What stands under it is left where it was.
func (f *DeckFile) RenameSection(at int, name string) error {
	body := []byte(f.doc.Body())
	heads := sectionHeads(body)
	if at < 0 || at >= len(heads) {
		return fmt.Errorf("%w: %d", ErrNoSuchSection, at)
	}
	line := headingLine(1, name)
	if heads[at].from < len(body) {
		line += "\n"
	}
	return f.doc.SpliceBody(heads[at].head, heads[at].from, line)
}

// RemoveSection takes away the heading of the section standing at one place in
// the deck. A section is a name and nothing else, so the cards that stood under
// it and what a person wrote beneath it stay where they are, under whatever now
// stands above them.
func (f *DeckFile) RemoveSection(at int) error {
	body := []byte(f.doc.Body())
	heads := sectionHeads(body)
	if at < 0 || at >= len(heads) {
		return fmt.Errorf("%w: %d", ErrNoSuchSection, at)
	}
	// The blank line under the heading went with it, so the block below closes
	// up against what the heading stood beneath.
	head, to := heads[at].head, heads[at].from
	for to < len(body) && (body[to] == '\n' || body[to] == '\r') {
		to++
	}
	// A section with nothing under it stood last, and what is above it now ends
	// on the one break a file ends with.
	written := ""
	if to == len(body) {
		head = trimmedEnd(body, 0, head)
		if head > 0 {
			written = "\n"
		}
	}
	return f.doc.SpliceBody(head, to, written)
}

// AddSection writes a section at the end of the deck. What a person wrote under
// its heading is written back under it, and nothing here reads it.
func (f *DeckFile) AddSection(s Section) error {
	body := []byte(f.doc.Body())
	blocks := []string{headingLine(SectionLevel, oneLine(s.Name))}
	if lead := trimBlankLines(markdown.Normalised(s.Lead)); lead != "" {
		blocks = append(blocks, lead)
	}
	at := len(body)
	return f.doc.SpliceBody(at, at, insert(body, at, strings.Join(blocks, "\n\n")))
}

// AddCard writes a card at the end of the deck.
func (f *DeckFile) AddCard(card Card) error {
	body := []byte(f.doc.Body())
	at := len(body)
	return f.doc.SpliceBody(at, at, insert(body, at, cardBlock(card)))
}

// AddCardUnder writes a card at the end of the section standing at one place in
// the deck, which is in front of the section after it. ErrNoSuchSection when
// the deck holds no section standing there.
func (f *DeckFile) AddCardUnder(at int, card Card) error {
	body := []byte(f.doc.Body())
	heads := sectionHeads(body)
	if at < 0 || at >= len(heads) {
		return fmt.Errorf("%w: %d", ErrNoSuchSection, at)
	}
	written := len(body)
	if at+1 < len(heads) {
		written = heads[at+1].head
	}
	return f.doc.SpliceBody(written, written, insert(body, written, cardBlock(card)))
}

// cardBlock is the markdown one card is written as: its heading, the wikilink
// naming the stencil it is cut by, the lead beneath it, and each value under a
// heading of its field's name.
func cardBlock(card Card) string {
	blocks := []string{headingLine(CardLevel, WriteHeading(oneLine(card.Heading), card.Mark))}
	if card.Stencil != "" {
		blocks = append(blocks, "[["+card.Stencil+"]]")
	}
	if lead := trimBlankLines(markdown.Normalised(card.Lead)); lead != "" {
		blocks = append(blocks, lead)
	}
	for _, v := range card.Values {
		blocks = append(blocks, headingLine(FieldLevel, oneLine(v.Field)))
		if text := trimBlankLines(markdown.Normalised(v.Text)); text != "" {
			blocks = append(blocks, text)
		}
	}
	return strings.Join(blocks, "\n\n")
}

// DeckBody is the markdown a deck is written as: the preamble as it arrived,
// each section and each card laid down in the order they stand, and the tail
// verbatim below the last value.
//
// Every card is laid down again, so a deck that goes through here comes back in
// the format's own spelling: one blank line between the blocks, and no
// whitespace at the end of a line, whatever the person wrote. A caller changing
// one card of a file somebody else writes by hand wants OpenDeckBody, which
// splices and leaves every card it did not reach alone.
//
// A card carries which section it stands under, so a section no card reaches is
// written where it stands: in front of the cards of the sections after it, or at
// the end where nothing stands under it. A card standing under a section the
// deck does not hold is ErrNoSuchSection.
func DeckBody(d Deck) (string, error) {
	scratch, err := OpenDeck(markdown.Create("", d.Preamble))
	if err != nil {
		return "", err
	}
	written := 0
	open := func(to int) error {
		for written < len(d.Sections) && written <= to {
			if err := scratch.AddSection(d.Sections[written]); err != nil {
				return err
			}
			written++
		}
		return nil
	}
	for _, card := range d.Cards {
		if card.Section != NoSection && (card.Section < 0 || card.Section >= len(d.Sections)) {
			return "", fmt.Errorf("%w: %d", ErrNoSuchSection, card.Section)
		}
		if err := open(card.Section); err != nil {
			return "", err
		}
		if err := scratch.AddCard(card); err != nil {
			return "", err
		}
	}
	if err := open(len(d.Sections)); err != nil {
		return "", err
	}

	body := scratch.doc.Body()
	if len(d.Cards) == 0 && len(d.Sections) == 0 {
		return body, nil
	}
	// The tail opens with the break that ends the last value, so the break the
	// last card was written with goes.
	return strings.TrimRight(body, "\n") + d.Tail, nil
}

// RenameField rewrites one field's heading in every card cut by the stencil
// filed at stencil, and reports how many it rewrote. The value under each
// heading is left as it was.
//
// Cutting is where each wikilink this deck writes lands, keyed by what stands
// in the brackets.
func (f *DeckFile) RenameField(cutting map[string]string, stencil, from, to string) (int, error) {
	body := []byte(f.doc.Body())
	deck, spans := readDeck(domain.Fingerprint{}, body)

	var heads []valueSpan
	for i, span := range spans {
		if !cutBy(deck.Cards[i], cutting, stencil) {
			continue
		}
		for _, v := range span.values {
			if v.field == from {
				heads = append(heads, v)
			}
		}
	}

	// Backwards, because a splice moves every byte after it.
	for i := len(heads) - 1; i >= 0; i-- {
		line := headingLine(FieldLevel, oneLine(to))
		if heads[i].from < len(body) {
			line += "\n"
		}
		if err := f.doc.SpliceBody(heads[i].head, heads[i].from, line); err != nil {
			return 0, err
		}
	}
	return len(heads), nil
}

// cutBy reports whether a card is cut by the stencil filed at stencil. The name
// under the heading is an ordinary wikilink, so what cuts the card is the note
// that name lands on.
func cutBy(card Card, cutting map[string]string, stencil string) bool {
	at, lands := cutting[card.Stencil]
	return lands && at == stencil
}
