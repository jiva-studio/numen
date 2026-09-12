package format

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// ErrNoSuchField is what changing a field says when the stencil declares no
// field of that name.
var ErrNoSuchField = errors.New("this stencil declares no field of that name")

// ErrFieldTaken is what renaming a field says when the stencil already declares
// a field of the name asked for. Two fields of one name are one field to every
// face and every card, and the values under the other are held by nothing.
var ErrFieldTaken = errors.New("this stencil already declares a field of that name")

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

// WriteIdentifier writes the identifier a file carrying none is to carry, and
// reports whether it wrote one.
func (f *StencilFile) WriteIdentifier(identifier string) (bool, error) {
	return writeIdentifier(f.doc, identifier)
}

// Stencil is what the file now says: the fields of the frontmatter in front of
// the writer, and the faces of the body in front of it.
func (f *StencilFile) Stencil(n domain.Note) Stencil {
	n.Body = f.doc.Body()
	n.Frontmatter = map[string]any{fieldsKey: readFieldNames(f.doc)}
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

// readFieldNames is the list of names under `fields` in the shape reading a
// stencil takes it.
func readFieldNames(doc *markdown.Document) []any {
	names, _ := doc.List(fieldsKey)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, name)
	}
	return out
}

// AddFace writes a face at the end of the stencil. A stencil shows a card once
// through each face it carries, so two faces of one name are two faces.
func (f *StencilFile) AddFace(face FaceTemplate) error {
	body := []byte(f.doc.Body())
	at := len(body)
	return f.doc.SpliceBody(at, at, insert(body, at, formatFace(face)))
}

// formatFace is the markdown one face is written as: its heading, the lead
// beneath it, and each side the face has under a heading of its name. A face
// missing a side is written missing it, and it is the face that lays out
// nothing.
func formatFace(face FaceTemplate) string {
	blocks := []string{headingLine(2, face.Name)}
	if lead := trimBlankLines(markdown.Normalised(face.Preamble)); lead != "" {
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

// StencilBody is the markdown these faces are written as, in the order they are
// to stand in the note: the preamble as it arrived, each face laid down by the
// format itself, and the tail verbatim below the last side.
//
// The faces are written into a stencil of no faces, one after another, so every
// face a caller gave stands in the file and two of one name are two faces.
func StencilBody(preamble string, faces []FaceTemplate, tail string) (string, error) {
	scratch, err := OpenStencil(markdown.Create("", preamble))
	if err != nil {
		return "", err
	}
	for _, face := range faces {
		if err := scratch.AddFace(face); err != nil {
			return "", err
		}
	}

	body := scratch.doc.Body()
	if len(faces) == 0 {
		return body, nil
	}
	// The tail opens with the break that ends the last side, so the break the
	// last face was written with goes.
	return strings.TrimRight(body, "\n") + tail, nil
}
