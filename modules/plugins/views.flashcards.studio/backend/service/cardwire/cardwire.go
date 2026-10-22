// Package cardwire provides wire conversions between flashcards domain models
// and protobuf representations.
package cardwire

import (
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cardid"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
)

// MaxStencils is how many stencils one answer carries. A vault holding more is
// answered with this many and told how many it holds, so a list that stops
// short says so.
const MaxStencils = 200

// NewWireStencil is one stencil as the schema carries it.
func NewWireStencil(path, title string, s format.Stencil) *v1.Stencil {
	out := &v1.Stencil{
		Path:     path,
		Title:    title,
		Fields:   s.Fields,
		Preamble: s.Preamble,
		Tail:     s.Tail,
		Faces:    make([]*v1.Face, 0, len(s.Faces)),
		Problems: newWireProblems(s.Problems),
	}
	for _, face := range s.Faces {
		out.Faces = append(out.Faces, newWireFace(face))
	}
	return out
}

// NewWireSummary is one stencil as the list of them names it.
func NewWireSummary(s cards.StencilSummary) *v1.StencilSummary {
	return &v1.StencilSummary{Path: s.Path, Title: s.Title, Fields: s.Fields}
}

// NewRenameResponse is what a rename reached and what it did not, as the schema carries
// it. A deck it could not be written to keeps the old heading, and the problem
// says which deck and why.
func NewRenameResponse(r cards.RenameResult, fingerprint *v1.Fingerprint) *v1.RenameStencilFieldResponse {
	out := &v1.RenameStencilFieldResponse{
		Decks: r.Decks,
		Cards: int32(r.Cards),
		At:    fingerprint,
	}
	for _, deck := range r.NotWritten {
		out.NotWritten = append(out.NotWritten, &v1.UnwrittenDeck{
			Path: deck.Path, Problem: newWireProblem(deck.Problem),
		})
	}
	return out
}

func newWireFace(f format.FaceTemplate) *v1.Face {
	return &v1.Face{Name: f.Name, Preamble: f.Preamble, Front: f.Front, Back: f.Back}
}

// NewWireDeck is one deck as the schema carries it. The cards go out in the order
// they stand in the file, and a problem's position is an index into that list.
func NewWireDeck(path, title string, d format.Deck, cutting map[string]string) *v1.Deck {
	out := &v1.Deck{
		Path:     path,
		Title:    title,
		Preamble: d.Preamble,
		Tail:     d.Tail,
		Sections: make([]*v1.Section, 0, len(d.Sections)),
		Cards:    make([]*v1.Card, 0, len(d.Cards)),
		Problems: newWireProblems(d.Problems),
	}
	for _, s := range d.Sections {
		out.Sections = append(out.Sections, &v1.Section{Name: s.Name, Preamble: s.Preamble})
	}
	for _, card := range d.Cards {
		out.Cards = append(out.Cards, newWireCard(card, cutting[card.StencilLink]))
	}
	return out
}

func newWireCard(c format.Card, pos string) *v1.Card {
	out := &v1.Card{
		Heading:      c.Heading,
		Mark:         string(c.Mark),
		SectionIndex: section(c.Section),
		StencilLink:  c.StencilLink,
		StencilPath:  pos,
		Preamble:     c.Preamble,
		Values:       make([]*v1.Value, 0, len(c.Values)),
	}
	for _, v := range c.Values {
		out.Values = append(out.Values, &v1.Value{Field: v.Field, Text: v.Text})
	}
	return out
}

// section maps the section index to a protobuf int32 pointer, returning nil for NoSection.
func section(pos int) *int32 {
	if pos == format.NoSection {
		return nil
	}
	index := int32(pos)
	return &index
}

// sectionOf maps a protobuf section pointer to domain section index, returning NoSection for nil.
func sectionOf(index *int32) int {
	if index == nil {
		return format.NoSection
	}
	return int(*index)
}

// newWireProblems is what was wrong with a file, in the order it was found. A check
// the schema names no fault for is drawn nowhere, and is left out.
func newWireProblems(problems []format.Problem) []*v1.Problem {
	if len(problems) == 0 {
		return nil
	}
	out := make([]*v1.Problem, 0, len(problems))
	for _, p := range problems {
		if one := newWireProblem(p); one != nil {
			out = append(out, one)
		}
	}
	return out
}

// newWireProblem is one problem as the schema carries it, and nothing for a fault
// the schema does not name.
func newWireProblem(p format.Problem) *v1.Problem {
	kind, isNamed := newWireProblemKind(p.Fault)
	if !isNamed {
		return nil
	}
	return &v1.Problem{
		Kind:  kind,
		Card:  position(p.Card),
		Face:  position(p.Face),
		Field: p.Field,
		Text:  p.Detail,
	}
}

// position converts an optional integer position to an int32 pointer.
func position(offset int) *int32 {
	if offset == format.NoPosition {
		return nil
	}
	pos := int32(offset)
	return &pos
}

// newWireProblemKind is what kind of problem it is, as the schema names it, and whether the schema
// names it at all.
func newWireProblemKind(fault format.Fault) (v1.ProblemKind, bool) {
	switch fault {
	case format.FaultTwoFields:
		return v1.ProblemKind_PROBLEM_KIND_FIELD_DECLARED_TWICE, true
	case format.FaultNoFields:
		return v1.ProblemKind_PROBLEM_KIND_STENCIL_WITHOUT_FIELDS, true
	case format.FaultFaceSide:
		return v1.ProblemKind_PROBLEM_KIND_FACE_MISSING_A_SIDE, true
	case format.FaultPlaceholder:
		return v1.ProblemKind_PROBLEM_KIND_PLACEHOLDER_UNDECLARED, true
	case format.FaultNoStencil:
		return v1.ProblemKind_PROBLEM_KIND_CARD_WITHOUT_A_STENCIL, true
	case format.FaultNotAStencil:
		return v1.ProblemKind_PROBLEM_KIND_STENCIL_IS_NOT_ONE, true
	case format.FaultTwoMarks:
		return v1.ProblemKind_PROBLEM_KIND_MARK_CARRIED_TWICE, true
	case format.FaultTwoValues:
		return v1.ProblemKind_PROBLEM_KIND_FIELD_WRITTEN_TWICE, true
	case format.FaultNotWritten:
		return v1.ProblemKind_PROBLEM_KIND_FIELD_NOT_RENAMED, true
	default:
		return v1.ProblemKind_PROBLEM_KIND_UNSPECIFIED, false
	}
}

// NewDeck is the deck a client is putting in the vault, in the words the
// core holds one in.
func NewDeck(w *v1.WriteDeckRequest) format.Deck {
	out := format.Deck{
		Preamble: w.GetPreamble(),
		Cards:    newCards(w.GetCards()),
		Tail:     w.GetTail(),
	}
	for _, s := range w.GetSections() {
		out.Sections = append(out.Sections,
			format.Section{Name: s.GetName(), Preamble: s.GetPreamble()})
	}
	return out
}

// newCards is the cards a client is putting into a deck, in the words the core
// holds them in.
func newCards(cs []*v1.Card) []format.Card {
	if len(cs) == 0 {
		return nil
	}
	out := make([]format.Card, 0, len(cs))
	for _, c := range cs {
		card := format.Card{
			Heading:     c.GetHeading(),
			Mark:        cardid.ID(c.GetMark()),
			Section:     sectionOf(c.SectionIndex),
			StencilLink: c.GetStencilLink(),
			Preamble:    c.GetPreamble(),
		}
		for _, v := range c.GetValues() {
			card.Values = append(card.Values, format.Value{Field: v.GetField(), Text: v.GetText()})
		}
		out = append(out, card)
	}
	return out
}

// NewFaces is the faces a client is putting into a stencil.
func NewFaces(fs []*v1.Face) []format.FaceTemplate {
	if len(fs) == 0 {
		return nil
	}
	out := make([]format.FaceTemplate, 0, len(fs))
	for _, f := range fs {
		out = append(out, format.FaceTemplate{
			Name: f.GetName(), Preamble: f.GetPreamble(), Front: f.GetFront(), Back: f.GetBack(),
		})
	}
	return out
}
