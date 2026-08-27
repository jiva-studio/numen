package webui

import (
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	format "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cards"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/cards"
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

func faceOf(f format.Face) *v1.Face {
	return &v1.Face{Name: f.Name, Front: f.Front, Back: f.Back}
}

// deckOf is one deck as the schema carries it. The cards go out in the order
// they stand in the file, and a problem's position is an index into that list.
func deckOf(path, title string, d format.Deck, cutting map[string]string) *v1.Deck {
	out := &v1.Deck{
		Path:     path,
		Title:    title,
		Preamble: d.Preamble,
		Tail:     d.Tail,
		Cards:    make([]*v1.Card, 0, len(d.Cards)),
		Problems: problemsOf(d.Problems),
	}
	for _, card := range d.Cards {
		out.Cards = append(out.Cards, cardOf(card, cutting[card.Stencil]))
	}
	return out
}

func cardOf(c format.Card, at string) *v1.Card {
	out := &v1.Card{
		Name:      c.Name,
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

// problemsOf is what was wrong with a file, in the order it was found. A check
// the schema names no fault for is drawn nowhere, and is left out.
func problemsOf(problems []format.Problem) []*v1.Problem {
	if len(problems) == 0 {
		return nil
	}
	out := make([]*v1.Problem, 0, len(problems))
	for _, p := range problems {
		fault, named := faultOf(p.Check)
		if !named {
			continue
		}
		out = append(out, &v1.Problem{
			Fault: fault,
			Card:  position(p.Card),
			Face:  position(p.Face),
			Field: p.Field,
			Text:  p.Detail,
		})
	}
	return out
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
func faultOf(check format.Check) (v1.Fault, bool) {
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
	case format.CheckNoName:
		return v1.Fault_FAULT_CARD_WITHOUT_A_NAME, true
	case format.CheckTwoCards:
		return v1.Fault_FAULT_CARD_NAMED_TWICE, true
	case format.CheckTwoValues:
		return v1.Fault_FAULT_FIELD_WRITTEN_TWICE, true
	case format.CheckFirstFieldTwice:
		return v1.Fault_FAULT_FIRST_FIELD_WRITTEN_TWICE, true
	case format.CheckNotWritten:
		return v1.Fault_FAULT_FIELD_NOT_RENAMED, true
	default:
		return v1.Fault_FAULT_UNSPECIFIED, false
	}
}

// cardsOf is the cards a client is putting into a deck, in the words the core
// holds them in.
func cardsOf(cs []*v1.Card) []format.Card {
	if len(cs) == 0 {
		return nil
	}
	out := make([]format.Card, 0, len(cs))
	for _, c := range cs {
		card := format.Card{Name: c.GetName(), Stencil: c.GetStencil(), Lead: c.GetLead()}
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
		out = append(out, format.Face{Name: f.GetName(), Front: f.GetFront(), Back: f.GetBack()})
	}
	return out
}
