package cards

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
)

// ErrNoSuchCard is what changing a card says when the deck holds no card of
// that name. A card is named by its heading, so a renamed heading is a
// different card.
var ErrNoSuchCard = errors.New("this deck holds no card of that name")

// ErrNoSuchField is what changing a field says when the stencil declares no
// field of that name.
var ErrNoSuchField = errors.New("this stencil declares no field of that name")

// ErrFieldTaken is what renaming a field says when the stencil already declares
// a field of the name asked for. Two fields of one name are one field to every
// face and every card, and the values under the other are held by nothing.
var ErrFieldTaken = errors.New("this stencil already declares a field of that name")

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

// Bytes is the deck as it now stands.
func (f *DeckFile) Bytes() []byte { return f.doc.Bytes() }

// Stamped writes the identifier a file carrying none is to carry, and reports
// whether it wrote one. The application changing what a note holds is what
// writes an identifier into it.
func (f *DeckFile) Stamped(identifier string) (bool, error) { return stamped(f.doc, identifier) }

// Deck is what the file now says.
func (f *DeckFile) Deck(ref domain.FileRef) Deck {
	deck, _ := readDeck(ref, []byte(f.doc.Body()))
	return deck
}

// SetValue writes what a card holds under one field. A field the card does not
// carry yet is added at the end of it, which leaves the fields it does carry in
// the order the person wrote them.
func (f *DeckFile) SetValue(card, field, value string) error {
	body := []byte(f.doc.Body())
	_, spans := readDeck(domain.FileRef{}, body)

	for _, span := range spans {
		if span.name != card {
			continue
		}
		for _, v := range span.values {
			if v.field == field {
				return f.doc.SpliceBody(v.from, v.to, under(value, v.to == len(body)))
			}
		}
		block := headingLine(3, field)
		if text := trimBlankLines(markdown.Normalised(value)); text != "" {
			block += "\n\n" + text
		}
		return f.doc.SpliceBody(span.end, span.end, insert(body, span.end, block))
	}
	return fmt.Errorf("%w: %s", ErrNoSuchCard, card)
}

// SetName writes what a card holds for its stencil's first field, which is the
// heading line and no heading below it. The card is addressed by the name it
// stands under now, and it is the card of the new name from here on.
func (f *DeckFile) SetName(card, name string) error {
	body := []byte(f.doc.Body())
	_, spans := readDeck(domain.FileRef{}, body)

	for _, span := range spans {
		if span.name != card {
			continue
		}
		line := headingLine(2, name)
		if span.from < len(body) {
			line += "\n"
		}
		return f.doc.SpliceBody(span.head, span.from, line)
	}
	return fmt.Errorf("%w: %s", ErrNoSuchCard, card)
}

// AddCard writes a card at the end of the deck.
func (f *DeckFile) AddCard(card Card) error {
	body := []byte(f.doc.Body())
	blocks := []string{headingLine(2, card.Name)}
	if card.Stencil != "" {
		blocks = append(blocks, "[["+card.Stencil+"]]")
	}
	if lead := trimBlankLines(markdown.Normalised(card.Lead)); lead != "" {
		blocks = append(blocks, lead)
	}
	for _, v := range card.Values {
		blocks = append(blocks, headingLine(3, v.Field))
		if text := trimBlankLines(markdown.Normalised(v.Text)); text != "" {
			blocks = append(blocks, text)
		}
	}
	at := len(body)
	return f.doc.SpliceBody(at, at, insert(body, at, strings.Join(blocks, "\n\n")))
}

// RenameField rewrites one field's heading in every card cut by the stencil
// filed at stencil, and reports how many it rewrote. The value under each
// heading is left as it was.
//
// Cutting is where each wikilink this deck writes lands, keyed by what stands
// in the brackets.
func (f *DeckFile) RenameField(cutting map[string]string, stencil, from, to string) (int, error) {
	body := []byte(f.doc.Body())
	deck, spans := readDeck(domain.FileRef{}, body)

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
		line := headingLine(3, to)
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

// StencilFile is a stencil held open, and what holds for a deck holds here: a
// face is changed by replacing the run it occupies.
type StencilFile struct {
	doc *markdown.Document
}

// OpenStencil reads a stencil for changing.
func OpenStencil(raw []byte) (*StencilFile, error) {
	doc, err := markdown.Open(raw)
	if err != nil {
		return nil, err
	}
	return &StencilFile{doc: doc}, nil
}

// Bytes is the stencil as it now stands.
func (f *StencilFile) Bytes() []byte { return f.doc.Bytes() }

// Stamped writes the identifier a file carrying none is to carry, and reports
// whether it wrote one.
func (f *StencilFile) Stamped(identifier string) (bool, error) { return stamped(f.doc, identifier) }

// stamped writes an identifier into a file that carries none.
func stamped(doc *markdown.Document, identifier string) (bool, error) {
	if _, carried := doc.Identifier(); carried {
		return false, nil
	}
	return true, doc.SetIdentifier(identifier)
}

// Stencil is what the file now says: the fields of the frontmatter in front of
// the writer, and the faces of the body in front of it.
func (f *StencilFile) Stencil(n domain.Note) Stencil {
	n.Body = f.doc.Body()
	n.Frontmatter = map[string]any{fieldsKey: declared(f.doc)}
	return ReadStencil(n)
}

// Fields is what the stencil now declares, in the order a person is asked for
// them.
func (f *StencilFile) Fields() []string {
	names, _ := f.doc.List(fieldsKey)
	return names
}

// SetFields writes the fields the stencil declares from now on. Every other key
// of the frontmatter is left as the bytes it arrived as.
func (f *StencilFile) SetFields(names []string) error {
	return f.doc.SetList(fieldsKey, names)
}

// RenameField gives one declared field a different name and leaves it where it
// stands in the order. ErrNoSuchField when the stencil declares no such field,
// and ErrFieldTaken when it already declares one of the name asked for.
//
// A field's name is written twice in this file: where `fields` declares it, and
// in the braces of every face that places it. Both are written here, so the
// stencil that comes out declares what its faces place.
func (f *StencilFile) RenameField(from, to string) error {
	names := f.Fields()
	at := slices.Index(names, from)
	if at < 0 {
		return fmt.Errorf("%w: %s", ErrNoSuchField, from)
	}
	if taken := slices.Index(names, to); taken >= 0 && taken != at {
		return fmt.Errorf("%w: %s", ErrFieldTaken, to)
	}
	names[at] = to
	if err := f.SetFields(names); err != nil {
		return err
	}
	return f.places(from, to)
}

// places writes the new name into every `{{Field}}` that named the old one. The
// markdown around the braces is the person's and is left as it was.
func (f *StencilFile) places(from, to string) error {
	body := f.doc.Body()
	written := "{{" + to + "}}"

	// Backwards, because a splice moves every byte after it.
	found := placeholderRe.FindAllStringSubmatchIndex(body, -1)
	for i := len(found) - 1; i >= 0; i-- {
		at := found[i]
		if body[at[2]:at[3]] != from {
			continue
		}
		if err := f.doc.SpliceBody(at[0], at[1], written); err != nil {
			return err
		}
	}
	return nil
}

// declared is the list of names under `fields` in the shape reading a stencil
// takes it.
func declared(doc *markdown.Document) []any {
	names, _ := doc.List(fieldsKey)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, name)
	}
	return out
}

// AddFace writes a face at the end of the stencil. A stencil shows a card once
// through each face it carries, so two faces of one name are two faces.
func (f *StencilFile) AddFace(face Face) error {
	body := []byte(f.doc.Body())
	at := len(body)
	return f.doc.SpliceBody(at, at, insert(body, at, laid(face)))
}

// laid is the markdown one face is written as: its heading, the lead beneath
// it, and each side the face has under a heading of its name. A face missing a
// side is written missing it, and it is the face that lays out nothing.
func laid(face Face) string {
	blocks := []string{headingLine(2, face.Name)}
	if lead := trimBlankLines(markdown.Normalised(face.Lead)); lead != "" {
		blocks = append(blocks, lead)
	}
	for _, side := range []struct{ heading, text string }{
		{frontHeading, face.Front},
		{backHeading, face.Back},
	} {
		text := trimBlankLines(markdown.Normalised(side.text))
		if text == "" {
			continue
		}
		blocks = append(blocks, headingLine(3, side.heading), text)
	}
	return strings.Join(blocks, "\n\n")
}

// headingLine is the line a card, a face, a field or a side stands under. A
// heading carrying no text is the hashes and nothing after them.
func headingLine(level int, name string) string {
	hashes := strings.Repeat("#", level)
	if name == "" {
		return hashes
	}
	return hashes + " " + name
}

// under is what stands beneath a heading: a blank line, the text, and a blank
// line after it. The last section of a file ends with the one break every file
// ends with.
func under(value string, last bool) string {
	text := trimBlankLines(markdown.Normalised(value))
	switch {
	case text == "" && last:
		return ""
	case text == "":
		return "\n"
	case last:
		return "\n" + text + "\n"
	}
	return "\n" + text + "\n\n"
}

// insert is a block written into the body at one byte, with the blank line
// above it that markdown wants and the blank line below it when something
// follows.
func insert(body []byte, at int, block string) string {
	written := gap(body, at) + block + "\n"
	if at < len(body) {
		written += "\n"
	}
	return written
}

// gap is the break and the blank line a new block needs, counting what already
// stands above it.
func gap(body []byte, at int) string {
	above := markdown.Normalised(string(body[:at]))
	switch {
	// Nothing but blank lines above is the top of the body, and a block written
	// there stands where it is.
	case strings.TrimSpace(above) == "" || strings.HasSuffix(above, "\n\n"):
		return ""
	case strings.HasSuffix(above, "\n"):
		return "\n"
	}
	return "\n\n"
}
