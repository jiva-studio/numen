package format

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardid"
)

// ErrNoSuchCard is returned when a change addresses a card id that no card of
// the deck carries. A card is identified by its card id, so a card rewritten
// in full keeps its identity.
var ErrNoSuchCard = errors.New("card not found in deck")

// ErrTwoCards is returned when a change addresses a card id carried by two
// cards of the deck. Both cards are read and displayed, and the change does
// not choose between them.
var ErrTwoCards = errors.New("two cards of this deck carry that card id")

// ErrNoSuchSection is returned when a change addresses a section index that
// the deck does not hold.
var ErrNoSuchSection = errors.New("section not found in deck")

// ErrNoSuchValue is returned when a value is removed from a card that has no
// heading for that field. The card model may declare the field; the card
// file does not contain it.
var ErrNoSuchValue = errors.New("this card writes no value under that field")

// DeckFile is a deck opened for changing. Every byte of the file that a
// change does not reach is unchanged. A change is a splice: the run one value
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

// Bytes is the deck with all changes applied.
func (f *DeckFile) Bytes() []byte { return f.doc.Bytes() }

// Body is the deck's prose with all changes applied.
func (f *DeckFile) Body() string { return f.doc.Body() }

// WriteIdentifier writes the identifier a file carrying none is to carry, and
// reports whether it wrote one. The application changing what a note holds is
// what writes an identifier into it.
func (f *DeckFile) WriteIdentifier(identifier string) (bool, error) {
	return writeIdentifier(f.doc, identifier)
}

// writeIdentifier writes an identifier into a file that carries none.
func writeIdentifier(doc *markdown.Document, identifier string) (bool, error) {
	if _, carried := doc.Identifier(); carried {
		return false, nil
	}
	return true, doc.SetIdentifier(identifier)
}

// Deck is the deck read from the current content of the file.
func (f *DeckFile) Deck(ref domain.FileStat) Deck {
	deck, _ := readDeck(ref, []byte(f.doc.Body()))
	return deck
}

// Whole makes every card of the file whole and reports the card ids it gave, which
// is what Whole does to a body. The file keeps its own line endings.
func (f *DeckFile) Whole(
	cardModels map[string]CardModel, createID func() cardid.ID,
) ([]CardID, error) {
	body, provided := Whole(markdown.Normalise(f.doc.Body()), cardModels, createID)
	if err := f.doc.SetBody(body); err != nil {
		return nil, err
	}
	return provided, nil
}

// SetValue writes the value of one field of the card with a card id. A field
// the card does not contain is added at the end of the card, and the fields it
// contains keep their order.
//
// A card id carried by two cards of the deck addresses neither: the result is
// ErrTwoCards.
//
// Writing the first field rewrites the heading, which is done where the deck
// is made whole and not here.
func (f *DeckFile) SetValue(card cardid.ID, field, value string) error {
	body := []byte(f.doc.Body())
	span, err := f.getCardSpan(body, card)
	if err != nil {
		return err
	}
	for _, v := range span.values {
		if v.field == field {
			return f.doc.SpliceBody(v.from, v.to, formatSection(value, v.to == len(body)))
		}
	}
	block := headingLine(FieldLevel, oneLine(field))
	if text := trimBlankLines(markdown.Normalise(value)); text != "" {
		block += "\n\n" + text
	}
	return f.doc.SpliceBody(span.end, span.end, insert(body, span.end, block))
}

// RemoveValue removes one field from a card: its heading line and its value.
// The other fields keep their order, and a card with no heading for that
// field yields ErrNoSuchValue.
//
// Removing the first field rewrites the heading, which is done where the deck
// is made whole and not here.
func (f *DeckFile) RemoveValue(card cardid.ID, field string) error {
	body := []byte(f.doc.Body())
	span, err := f.getCardSpan(body, card)
	if err != nil {
		return err
	}
	for _, v := range span.values {
		if v.field != field {
			continue
		}
		// The blank line after the value is removed with it, so the following block
		// adjoins the preceding one.
		head, to := v.head, v.to
		content := ""
		// A value at the end of the file leaves the preceding content ending with
		// the single line break a file ends with.
		if to == len(body) {
			head = getTrimmedEnd(body, 0, head)
			if head > 0 {
				content = "\n"
			}
		}
		return f.doc.SpliceBody(head, to, content)
	}
	return fmt.Errorf("%w: %s", ErrNoSuchValue, field)
}

// SetCardModel sets the card model link of a card. A card without a link
// receives the link as the paragraph after its heading, and the values it
// contains keep their position.
func (f *DeckFile) SetCardModel(card cardid.ID, name string) error {
	body := []byte(f.doc.Body())
	span, err := f.getCardSpan(body, card)
	if err != nil {
		return err
	}
	link := "[[" + oneLine(name) + "]]"
	if span.linkTo > span.linkFrom {
		return f.doc.SpliceBody(span.linkFrom, span.linkTo, link)
	}
	return f.doc.SpliceBody(span.from, span.from, "\n"+link+"\n")
}

// RemoveCard removes a card from the deck: its heading, its card model link
// and every value. The content before and after the card is unchanged.
func (f *DeckFile) RemoveCard(card cardid.ID) error {
	body := []byte(f.doc.Body())
	span, err := f.getCardSpan(body, card)
	if err != nil {
		return err
	}
	// The blank line after the card is removed with it, so the following block
	// adjoins the preceding one.
	head, to := span.head, span.end
	content := ""
	// A card at the end of the file leaves the preceding content ending with the
	// single line break a file ends with.
	if to == len(body) {
		head = getTrimmedEnd(body, 0, head)
		if head > 0 {
			content = "\n"
		}
	}
	return f.doc.SpliceBody(head, to, content)
}

// getCardSpan returns the span of the card with a card id. A deck with no card
// of that card id yields ErrNoSuchCard, and a deck with two yields
// ErrTwoCards.
func (f *DeckFile) getCardSpan(body []byte, card cardid.ID) (cardSpan, error) {
	_, spans := readDeck(createEmptyStat(), body)
	// gomutants:disable-next-line INTEGER_INCREMENT reason="at is only compared with zero, so -1 becomes -2 and nothing reads the difference"
	pos := -1
	for i, span := range spans {
		if span.cardID == "" || span.cardID != card {
			continue
		}
		if pos >= 0 {
			return cardSpan{}, fmt.Errorf("%w: %s", ErrTwoCards, card)
		}
		pos = i
	}
	if pos < 0 {
		return cardSpan{}, fmt.Errorf("%w: %s", ErrNoSuchCard, card)
	}
	return spans[pos], nil
}

// RenameSection gives the section standing at one place in the deck another
// name. What is placed under it is left where it was.
func (f *DeckFile) RenameSection(index int, name string) error {
	body := []byte(f.doc.Body())
	heads := sectionHeads(body)
	if index < 0 || index >= len(heads) {
		return fmt.Errorf("%w: %d", ErrNoSuchSection, index)
	}
	line := headingLine(1, name)
	if from := heads[index].from; from > 0 && body[from-1] == '\n' {
		line += "\n"
	}
	return f.doc.SpliceBody(heads[index].head, heads[index].from, line)
}

// RemoveSection removes the heading of the section at one index. A section is
// a name only, so the cards and prose after the heading are unchanged and
// belong to the preceding section.
func (f *DeckFile) RemoveSection(index int) error {
	body := []byte(f.doc.Body())
	heads := sectionHeads(body)
	if index < 0 || index >= len(heads) {
		return fmt.Errorf("%w: %d", ErrNoSuchSection, index)
	}
	// The blank line after the heading is removed with it, so the following block
	// adjoins the preceding one.
	head, to := heads[index].head, heads[index].from
	for to < len(body) && (body[to] == '\n' || body[to] == '\r') {
		to++
	}
	// A section at the end of the file leaves the preceding content ending with
	// the single line break a file ends with.
	content := ""
	if to == len(body) {
		head = getTrimmedEnd(body, 0, head)
		if head > 0 {
			content = "\n"
		}
	}
	return f.doc.SpliceBody(head, to, content)
}

// AddSection appends a section at the end of the deck. The prose after the
// heading is written unchanged.
func (f *DeckFile) AddSection(s Section) error {
	body := []byte(f.doc.Body())
	blocks := []string{headingLine(SectionLevel, oneLine(s.Name))}
	if lead := trimBlankLines(markdown.Normalise(s.Preamble)); lead != "" {
		blocks = append(blocks, lead)
	}
	pos := len(body)
	return f.doc.SpliceBody(pos, pos, insert(body, pos, strings.Join(blocks, "\n\n")))
}

// AddCard writes a card at the end of the deck.
func (f *DeckFile) AddCard(card Card) error {
	body := []byte(f.doc.Body())
	pos := len(body)
	return f.doc.SpliceBody(pos, pos, insert(body, pos, cardBlock(card)))
}

// AddCardUnder writes a card at the end of the section standing at one place in
// the deck, which is in front of the section after it. ErrNoSuchSection when
// the deck holds no section standing there.
func (f *DeckFile) AddCardUnder(sectionIndex int, card Card) error {
	body := []byte(f.doc.Body())
	heads := sectionHeads(body)
	if sectionIndex < 0 || sectionIndex >= len(heads) {
		return fmt.Errorf("%w: %d", ErrNoSuchSection, sectionIndex)
	}
	content := len(body)
	if sectionIndex+1 < len(heads) {
		content = heads[sectionIndex+1].head
	}
	return f.doc.SpliceBody(content, content, insert(body, content, cardBlock(card)))
}

// cardBlock is the markdown one card is written as: its heading, the wikilink
// naming the card model it is cut by, the lead beneath it, and each value under a
// heading of its field's name.
func cardBlock(card Card) string {
	blocks := []string{headingLine(CardLevel, WriteHeading(oneLine(card.Heading), card.CardID))}
	if card.CardModelLink != "" {
		blocks = append(blocks, "[["+card.CardModelLink+"]]")
	}
	if lead := trimBlankLines(markdown.Normalise(card.Preamble)); lead != "" {
		blocks = append(blocks, lead)
	}
	for _, v := range card.Values {
		blocks = append(blocks, headingLine(FieldLevel, oneLine(v.Field)))
		if text := trimBlankLines(markdown.Normalise(v.Text)); text != "" {
			blocks = append(blocks, text)
		}
	}
	return strings.Join(blocks, "\n\n")
}

// DeckBody is the markdown a deck is written as: the preamble unchanged,
// each section and each card in deck order, and the tail verbatim after the
// last value.
//
// Every card is written again, so the output uses the format's own spelling:
// one blank line between blocks and no trailing whitespace on a line. A
// caller changing one card of a file edited manually uses OpenDeckBody, which
// splices and leaves every card it does not change unchanged.
//
// A card names the section it belongs to, so a section no card belongs to is
// written at its position: before the cards of the following sections, or at
// the end when no card follows it. A card whose section index the deck does
// not hold yields ErrNoSuchSection.
type deckBodyWriter struct {
	scratch      *DeckFile
	sections     []Section
	sectionCount int
}

func (w *deckBodyWriter) writeSectionsUpTo(to int) error {
	for w.sectionCount < len(w.sections) && w.sectionCount <= to {
		if err := w.scratch.AddSection(w.sections[w.sectionCount]); err != nil {
			return err
		}
		w.sectionCount++
	}
	return nil
}

func (w *deckBodyWriter) writeCard(card Card) error {
	// gomutants:disable-next-line INTEGER_DECREMENT reason="0 becomes -1, and the one place the two differ is NoSection, which the first condition already excludes"
	if card.Section != NoSection && (card.Section < 0 || card.Section >= len(w.sections)) {
		return fmt.Errorf("%w: %d", ErrNoSuchSection, card.Section)
	}
	if err := w.writeSectionsUpTo(card.Section); err != nil {
		return err
	}
	return w.scratch.AddCard(card)
}

// DeckBody renders the cards and sections of d as the body of a deck note.
func DeckBody(d Deck) (string, error) {
	scratch, err := OpenDeck(markdown.Create("", d.Preamble))
	if err != nil {
		return "", err
	}
	w := deckBodyWriter{
		scratch:      scratch,
		sections:     d.Sections,
		sectionCount: 0,
	}
	for _, card := range d.Cards {
		if err := w.writeCard(card); err != nil {
			return "", err
		}
	}
	if err := w.writeSectionsUpTo(len(d.Sections)); err != nil {
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

// RenameField rewrites one field's heading in every card cut by the card model
// at cardModel, and returns the number of cards rewritten. The value under
// each heading is unchanged.
//
// cutting maps each wikilink name written in the deck to the path of the note
// it resolves to.
func (f *DeckFile) RenameField(cutting map[string]string, cardModel, from, to string) (int, error) {
	body := []byte(f.doc.Body())
	deck, spans := readDeck(createEmptyStat(), body)

	var heads []valueSpan
	for i, span := range spans {
		if !usesCardModel(deck.Cards[i], cutting, cardModel) {
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

// usesCardModel reports whether a card is cut by the card model filed at card model. The name
// under the heading is an ordinary wikilink, so what cuts the card is the note
// that name lands on.
func usesCardModel(card Card, linkTargets map[string]string, cardModel string) bool {
	pos, lands := linkTargets[card.CardModelLink]
	return lands && pos == cardModel
}
