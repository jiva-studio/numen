package webui

import (
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	format "github.com/jiva-studio/numen/modules/libs/core/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
)

// maxStencils is how many stencils one answer carries. A vault holding more is
// answered with this many and told how many it holds, so a list that stops
// short says so.
const maxStencils = 200

// stencilOf is one stencil as the schema carries it.
func stencilOf(path, title string, s format.Stencil) *v1.Stencil {
	out := &v1.Stencil{
		Path:     path,
		Title:    title,
		Fields:   s.Fields,
		Preamble: s.Preamble,
		Tail:     s.Tail,
		Faces:    make([]*v1.Face, 0, len(s.Faces)),
		Problems: problemsOf(s.Problems),
	}
	for _, face := range s.Faces {
		out.Faces = append(out.Faces, faceOf(face))
	}
	return out
}

// offeredOf is one stencil as the list of them names it.
func offeredOf(s cards.Listed) *v1.Offered {
	return &v1.Offered{Path: s.Path, Title: s.Title, Fields: s.Fields}
}

// renamedOf is what a rename reached and what it did not, as the schema carries
// it. A deck it could not be written to keeps the old heading, and the problem
// says which deck and why.
func renamedOf(r cards.Renamed) *v1.RenameFieldResponse {
	out := &v1.RenameFieldResponse{
		Decks: r.Decks,
		Cards: int32(r.Cards),
		At:    fingerprintOf(r.Stencil),
	}
	for _, deck := range r.NotWritten {
		out.NotWritten = append(out.NotWritten, &v1.NotWritten{
			Path: deck.Path, Problem: problemOf(deck.Problem),
		})
	}
	return out
}

func faceOf(f format.Face) *v1.Face {
	return &v1.Face{Name: f.Name, Lead: f.Lead, Front: f.Front, Back: f.Back}
}

// deckOf is one deck as the schema carries it. The cards go out in the order
// they stand in the file, and a problem's position is an index into that list.
func deckOf(path, title string, d format.Deck, cutting map[string]string) *v1.Deck {
	out := &v1.Deck{
		Path:     path,
		Title:    title,
		Preamble: d.Preamble,
		Tail:     d.Tail,
		Sections: make([]*v1.Section, 0, len(d.Sections)),
		Cards:    make([]*v1.Card, 0, len(d.Cards)),
		Problems: problemsOf(d.Problems),
	}
	for _, s := range d.Sections {
		out.Sections = append(out.Sections, &v1.Section{Name: s.Name, Lead: s.Lead})
	}
	for _, card := range d.Cards {
		out.Cards = append(out.Cards, cardOf(card, cutting[card.Stencil]))
	}
	return out
}

func cardOf(c format.Card, at string) *v1.Card {
	out := &v1.Card{
		Heading:   c.Heading,
		Mark:      c.Mark,
		Section:   section(c.Section),
		Stencil:   c.Stencil,
		StencilAt: at,
		Lead:      c.Lead,
		Values:    make([]*v1.Value, 0, len(c.Values)),
	}
	for _, v := range c.Values {
		out.Values = append(out.Values, &v1.Value{Field: v.Field, Text: v.Text})
	}
	return out
}

// section is where a card's section stands in the deck's own, as the schema
// carries it. A card standing before the first section carries nothing.
func section(at int) *int32 {
	if at == format.NoSection {
		return nil
	}
	under := int32(at)
	return &under
}

// sectionOf is where a card the client is writing stands, in the words the core
// holds it in. A card the client left without one stands before the first
// section.
func sectionOf(under *int32) int {
	if under == nil {
		return format.NoSection
	}
	return int(*under)
}

// problemsOf is what was wrong with a file, in the order it was found. A check
// the schema names no fault for is drawn nowhere, and is left out.
func problemsOf(problems []format.Problem) []*v1.Problem {
	if len(problems) == 0 {
		return nil
	}
	out := make([]*v1.Problem, 0, len(problems))
	for _, p := range problems {
		if one := problemOf(p); one != nil {
			out = append(out, one)
		}
	}
	return out
}

// problemOf is one problem as the schema carries it, and nothing for a check
// the schema names no fault for.
func problemOf(p format.Problem) *v1.Problem {
	fault, named := faultOf(p.Check)
	if !named {
		return nil
	}
	return &v1.Problem{
		Fault: fault,
		Card:  position(p.Card),
		Face:  position(p.Face),
		Field: p.Field,
		Text:  p.Detail,
	}
}

// position is where a problem stands, as the schema carries it. A problem about
// no card and no face carries nothing.
func position(at int) *int32 {
	if at == format.NoPosition {
		return nil
	}
	stands := int32(at)
	return &stands
}

// faultOf is what a problem is, as the schema names it, and whether the schema
// names it at all.
func faultOf(check format.Fault) (v1.Fault, bool) {
	switch check {
	case format.CheckTwoFields:
		return v1.Fault_FAULT_FIELD_DECLARED_TWICE, true
	case format.CheckNoFields:
		return v1.Fault_FAULT_STENCIL_WITHOUT_FIELDS, true
	case format.CheckFaceSide:
		return v1.Fault_FAULT_FACE_MISSING_A_SIDE, true
	case format.CheckPlaceholder:
		return v1.Fault_FAULT_PLACEHOLDER_UNDECLARED, true
	case format.CheckNoStencil:
		return v1.Fault_FAULT_CARD_WITHOUT_A_STENCIL, true
	case format.CheckNotAStencil:
		return v1.Fault_FAULT_STENCIL_IS_NOT_ONE, true
	case format.CheckTwoMarks:
		return v1.Fault_FAULT_MARK_CARRIED_TWICE, true
	case format.CheckTwoValues:
		return v1.Fault_FAULT_FIELD_WRITTEN_TWICE, true
	case format.CheckNotWritten:
		return v1.Fault_FAULT_FIELD_NOT_RENAMED, true
	default:
		return v1.Fault_FAULT_UNSPECIFIED, false
	}
}

// writtenDeck is the deck a client is putting in the vault, in the words the
// core holds one in.
func writtenDeck(w *v1.WriteDeckRequest) format.Deck {
	out := format.Deck{
		Preamble: w.GetPreamble(),
		Cards:    cardsOf(w.GetCards()),
		Tail:     w.GetTail(),
	}
	for _, s := range w.GetSections() {
		out.Sections = append(out.Sections, format.Section{Name: s.GetName(), Lead: s.GetLead()})
	}
	return out
}

// cardsOf is the cards a client is putting into a deck, in the words the core
// holds them in.
func cardsOf(cs []*v1.Card) []format.Card {
	if len(cs) == 0 {
		return nil
	}
	out := make([]format.Card, 0, len(cs))
	for _, c := range cs {
		card := format.Card{
			Heading: c.GetHeading(),
			Mark:    c.GetMark(),
			Section: sectionOf(c.Section),
			Stencil: c.GetStencil(),
			Lead:    c.GetLead(),
		}
		for _, v := range c.GetValues() {
			card.Values = append(card.Values, format.Value{Field: v.GetField(), Text: v.GetText()})
		}
		out = append(out, card)
	}
	return out
}

// facesOf is the faces a client is putting into a stencil.
func facesOf(fs []*v1.Face) []format.Face {
	if len(fs) == 0 {
		return nil
	}
	out := make([]format.Face, 0, len(fs))
	for _, f := range fs {
		out = append(out, format.Face{
			Name: f.GetName(), Lead: f.GetLead(), Front: f.GetFront(), Back: f.GetBack(),
		})
	}
	return out
}
