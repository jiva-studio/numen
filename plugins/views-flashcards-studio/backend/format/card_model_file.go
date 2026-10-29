package format

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
)

// ErrNoSuchField is returned when changing a field of a card model that does not
// declare a field of that name.
var ErrNoSuchField = errors.New("this card model declares no field of that name")

// ErrFieldTaken is returned when renaming a field to a name that the card model
// already declares. Two fields with one name are one field to every template and
// every card, and the values under the other field would be orphaned.
var ErrFieldTaken = errors.New("this card model already declares a field of that name")

// CardModelFile is a card model note opened for editing; each edit replaces the
// run of text it targets.
type CardModelFile struct {
	doc *markdown.Document
}

// OpenCardModel reads a card model for changing.
func OpenCardModel(raw []byte) (*CardModelFile, error) {
	doc, err := markdown.Open(raw)
	if err != nil {
		return nil, err
	}
	return &CardModelFile{doc: doc}, nil
}

// Bytes returns the current content of the card model.
func (f *CardModelFile) Bytes() []byte { return f.doc.Bytes() }

// WriteIdentifier writes the identifier a file carrying none is to carry, and
// reports whether it wrote one.
func (f *CardModelFile) WriteIdentifier(identifier string) (bool, error) {
	return writeIdentifier(f.doc, identifier)
}

// Fields returns the fields the card model currently declares, in declaration
// order.
func (f *CardModelFile) Fields() []string {
	names, _ := f.doc.List(fieldsKey)
	return names
}

// SetFields writes the fields the card model declares from now on. Every other key
// of the frontmatter is left as the bytes it arrived as.
func (f *CardModelFile) SetFields(names []string) error {
	return f.doc.SetList(fieldsKey, names)
}

// RenameField renames one declared field in place. It returns ErrNoSuchField when
// the card model does not declare `from`, and ErrFieldTaken when it already
// declares `to`.
//
// A field's name is written twice in this file: where `fields` declares it, and
// in the braces of every template that places it. Both are written here, so the
// resulting card model declares what its templates place.
func (f *CardModelFile) RenameField(from, to string) error {
	names := f.Fields()
	pos := slices.Index(names, from)
	if pos < 0 {
		return fmt.Errorf("%w: %s", ErrNoSuchField, from)
	}
	if targetIndex := slices.Index(names, to); targetIndex >= 0 && targetIndex != pos {
		return fmt.Errorf("%w: %s", ErrFieldTaken, to)
	}
	names[pos] = to
	if err := f.SetFields(names); err != nil {
		return err
	}
	return f.replacePlaceholders(from, to)
}

// replacePlaceholders replaces the old name with the new name in every
// `{{Field}}` placeholder. The markdown around the braces is user content and is
// unchanged.
func (f *CardModelFile) replacePlaceholders(from, to string) error {
	body := f.doc.Body()
	content := "{{" + to + "}}"

	// Backwards, because a splice moves every byte after it.
	found := placeholderRe.FindAllStringSubmatchIndex(body, -1)
	for i := len(found) - 1; i >= 0; i-- {
		pos := found[i]
		if body[pos[2]:pos[3]] != from {
			continue
		}
		if err := f.doc.SpliceBody(pos[0], pos[1], content); err != nil {
			return err
		}
	}
	return nil
}

// AddTemplate writes a template at the end of the card model. A card model shows a card once
// through each template it carries, so two templates of one name are two templates.
func (f *CardModelFile) AddTemplate(template CardTemplate) error {
	body := []byte(f.doc.Body())
	pos := len(body)
	return f.doc.SpliceBody(pos, pos, insert(body, pos, formatTemplate(template)))
}

// formatTemplate is the markdown one template is written as: its heading, the lead
// beneath it, and each side the template has under a heading of its name. A template
// missing a side is written missing it, and it is the template that lays out
// nothing.
func formatTemplate(template CardTemplate) string {
	blocks := []string{headingLine(2, template.Name)}
	if lead := trimBlankLines(markdown.Normalise(template.Preamble)); lead != "" {
		blocks = append(blocks, lead)
	}
	for _, side := range []struct{ heading, text string }{
		{frontHeading, template.Front},
		{backHeading, template.Back},
	} {
		text := trimBlankLines(markdown.Normalise(side.text))
		if text == "" {
			continue
		}
		blocks = append(blocks, headingLine(3, side.heading), text)
	}
	return strings.Join(blocks, "\n\n")
}

// CardModelBody is the markdown these templates are written as, in the order they are
// to stand in the note: the preamble as it arrived, each template laid down by the
// format itself, and the tail verbatim below the last side.
//
// The templates are written into a card model of no templates, one after another, so every
func CardModelBody(preamble string, templates []CardTemplate, tail string) (string, error) {
	scratch, err := OpenCardModel(markdown.Create("", preamble))
	if err != nil {
		return "", err
	}
	for _, template := range templates {
		if err := scratch.AddTemplate(template); err != nil {
			return "", err
		}
	}

	body := scratch.doc.Body()
	if len(templates) == 0 {
		return body, nil
	}
	// The tail opens with the break that ends the last side, so the break the
	// last template was written with goes.
	return strings.TrimRight(body, "\n") + tail, nil
}
