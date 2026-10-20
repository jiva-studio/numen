package cardwire

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/cardid"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/format"
)

func ptr(n int32) *int32 { return &n }

func requireEqual(t *testing.T, got, want proto.Message) {
	t.Helper()
	if !proto.Equal(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestAStencilGoesOutWithItsFacesAndProblems(t *testing.T) {
	got := NewWireStencil("stencils/basic.md", "Basic", format.Stencil{
		Fields:   []string{"Front", "Back"},
		Preamble: "above",
		Tail:     "below",
		Faces: []format.FaceTemplate{
			{Name: "One", Preamble: "p1", Front: "f1", Back: "b1"},
			{Name: "Two", Preamble: "p2", Front: "f2", Back: "b2"},
		},
		Problems: []format.Problem{{Fault: format.FaultNoFields, Card: format.NoPosition, Face: 1, Field: "Front", Detail: "d"}},
	})
	requireEqual(t, got, &v1.Stencil{
		Path: "stencils/basic.md", Title: "Basic", Fields: []string{"Front", "Back"},
		Preamble: "above", Tail: "below",
		Faces: []*v1.Face{
			{Name: "One", Preamble: "p1", Front: "f1", Back: "b1"},
			{Name: "Two", Preamble: "p2", Front: "f2", Back: "b2"},
		},
		Problems: []*v1.Problem{{Kind: v1.ProblemKind_PROBLEM_KIND_STENCIL_WITHOUT_FIELDS, Face: ptr(1), Field: "Front", Text: "d"}},
	})
}

func TestAStencilWithoutProblemsCarriesNone(t *testing.T) {
	if got := NewWireStencil("p", "t", format.Stencil{}); got.Problems != nil || got.Faces == nil {
		t.Errorf("got problems %v, faces %v", got.Problems, got.Faces)
	}
	one := newWireProblems([]format.Problem{{Fault: format.FaultNoFields, Card: format.NoPosition, Face: format.NoPosition}})
	if len(one) != 1 {
		t.Errorf("a single problem went out as %d", len(one))
	}
	if newWireProblems(nil) != nil {
		t.Error("no problems went out as a list")
	}
}

func TestAStencilSummaryGoesOutWhole(t *testing.T) {
	got := NewWireSummary(cards.StencilSummary{Path: "p", Title: "t", Fields: []string{"a", "b"}})
	requireEqual(t, got, &v1.StencilSummary{Path: "p", Title: "t", Fields: []string{"a", "b"}})
}

func TestARenameGoesOutWithEveryDeckItCouldNotWrite(t *testing.T) {
	at := &v1.Fingerprint{}
	got := NewRenameResponse(cards.RenameResult{
		Decks: []string{"a.md"},
		Cards: 7,
		NotWritten: []cards.UnwrittenDeck{
			{Path: "x.md", Problem: format.Problem{Fault: format.FaultTwoFields, Card: format.NoPosition, Face: format.NoPosition}},
			{Path: "y.md", Problem: format.Problem{Fault: format.FaultNoFields, Card: 2, Face: format.NoPosition}},
		},
	}, at)
	requireEqual(t, got, &v1.RenameStencilFieldResponse{
		Decks: []string{"a.md"}, Cards: 7, At: at,
		NotWritten: []*v1.UnwrittenDeck{
			{Path: "x.md", Problem: &v1.Problem{Kind: v1.ProblemKind_PROBLEM_KIND_FIELD_DECLARED_TWICE}},
			{Path: "y.md", Problem: &v1.Problem{Kind: v1.ProblemKind_PROBLEM_KIND_STENCIL_WITHOUT_FIELDS, Card: ptr(2)}},
		},
	})
}

func TestADeckGoesOutWithItsSectionsCardsAndValues(t *testing.T) {
	got := NewWireDeck("decks/a.md", "A", format.Deck{
		Preamble: "above", Tail: "below",
		Sections: []format.Section{{Name: "S1", Preamble: "sp1"}, {Name: "S2", Preamble: "sp2"}},
		Cards: []format.Card{
			{Heading: "h1", Mark: cardid.ID("m1"), Section: format.NoSection, StencilLink: "basic", Preamble: "c1",
				Values: []format.Value{{Field: "Front", Text: "q"}, {Field: "Back", Text: "a"}}},
			{Heading: "h2", Section: 1, StencilLink: "other"},
		},
	}, map[string]string{"basic": "stencils/basic.md"})
	requireEqual(t, got, &v1.Deck{
		Path: "decks/a.md", Title: "A", Preamble: "above", Tail: "below",
		Sections: []*v1.Section{{Name: "S1", Preamble: "sp1"}, {Name: "S2", Preamble: "sp2"}},
		Cards: []*v1.Card{
			{Heading: "h1", Mark: "m1", StencilLink: "basic", StencilPath: "stencils/basic.md", Preamble: "c1",
				Values: []*v1.Value{{Field: "Front", Text: "q"}, {Field: "Back", Text: "a"}}},
			{Heading: "h2", SectionIndex: ptr(1), StencilLink: "other", Values: []*v1.Value{}},
		},
	})
}

func TestADeckWithNothingInItGoesOutWithEmptyLists(t *testing.T) {
	got := NewWireDeck("p", "t", format.Deck{}, nil)
	if got.Sections == nil || got.Cards == nil || got.Problems != nil {
		t.Errorf("sections %v, cards %v, problems %v", got.Sections, got.Cards, got.Problems)
	}
}

func TestADeckComesInWithItsSectionsCardsAndValues(t *testing.T) {
	got := NewDeck(&v1.WriteDeckRequest{
		Preamble: "above", Tail: "below",
		Sections: []*v1.Section{{Name: "S1", Preamble: "sp1"}, {Name: "S2", Preamble: "sp2"}},
		Cards: []*v1.Card{
			{Heading: "h1", Mark: "m1", SectionIndex: ptr(1), StencilLink: "basic", Preamble: "c1",
				Values: []*v1.Value{{Field: "Front", Text: "q"}, {Field: "Back", Text: "a"}}},
			{Heading: "h2"},
		},
	})
	want := format.Deck{
		Preamble: "above", Tail: "below",
		Sections: []format.Section{{Name: "S1", Preamble: "sp1"}, {Name: "S2", Preamble: "sp2"}},
		Cards: []format.Card{
			{Heading: "h1", Mark: cardid.ID("m1"), Section: 1, StencilLink: "basic", Preamble: "c1",
				Values: []format.Value{{Field: "Front", Text: "q"}, {Field: "Back", Text: "a"}}},
			{Heading: "h2", Section: format.NoSection},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestADeckComesInWithoutCardsOrSections(t *testing.T) {
	got := NewDeck(&v1.WriteDeckRequest{})
	if got.Cards != nil || got.Sections != nil {
		t.Errorf("cards %v, sections %v", got.Cards, got.Sections)
	}
	one := NewDeck(&v1.WriteDeckRequest{Cards: []*v1.Card{{Heading: "h"}}})
	if len(one.Cards) != 1 {
		t.Errorf("a single card came in as %d", len(one.Cards))
	}
}

func TestFacesComeIn(t *testing.T) {
	got := NewFaces([]*v1.Face{
		{Name: "One", Preamble: "p1", Front: "f1", Back: "b1"},
		{Name: "Two", Preamble: "p2", Front: "f2", Back: "b2"},
	})
	want := []format.FaceTemplate{
		{Name: "One", Preamble: "p1", Front: "f1", Back: "b1"},
		{Name: "Two", Preamble: "p2", Front: "f2", Back: "b2"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
	if NewFaces(nil) != nil {
		t.Error("no faces came in as a list")
	}
	if len(NewFaces([]*v1.Face{{Name: "x"}})) != 1 {
		t.Error("a single face did not come in")
	}
}

func TestASectionIsAnIndexOrNothing(t *testing.T) {
	if section(format.NoSection) != nil {
		t.Error("no section went out as an index")
	}
	if got := section(0); got == nil || *got != 0 {
		t.Errorf("section 0 went out as %v", got)
	}
	if sectionOf(nil) != format.NoSection {
		t.Error("an absent index came in as a section")
	}
	if got := sectionOf(ptr(0)); got != 0 {
		t.Errorf("index 0 came in as %d", got)
	}
}

func TestAPositionIsAnIndexOrNothing(t *testing.T) {
	if position(format.NoPosition) != nil {
		t.Error("no position went out as an index")
	}
	if got := position(0); got == nil || *got != 0 {
		t.Errorf("position 0 went out as %v", got)
	}
}
