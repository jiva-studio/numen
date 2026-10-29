// Package cardwire provides wire conversions between flashcards domain models
// and API messages.
package cardwire

import (
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

// MaxCardModels is how many card models one response carries. A vault holding
// more returns this many together with the total it holds, so a truncated list
// is reported as truncated.
const MaxCardModels = 200

// NewWireCardModel is one card model as the schema carries it.
func NewWireCardModel(path, title string, s format.CardModel) *api.CardModel {
	out := &api.CardModel{
		Path:      path,
		Title:     title,
		Fields:    s.Fields,
		Preamble:  s.Preamble,
		Tail:      s.Tail,
		Templates: make([]*api.CardTemplate, 0, len(s.Templates)),
		Problems:  newWireProblems(s.Problems),
	}
	for _, template := range s.Templates {
		out.Templates = append(out.Templates, newWireTemplate(template))
	}
	return out
}

// NewWireSummary is one card model as the list of them names it.
func NewWireSummary(s cards.CardModelSummary) *api.CardModelSummary {
	return &api.CardModelSummary{Path: s.Path, Title: s.Title, Fields: s.Fields}
}

// NewRenameResponse reports which decks a rename reached and which it did not,
// as the schema carries it. A deck that could not be written keeps the old
// heading, and the problem names the deck and the reason.
func NewRenameResponse(r cards.RenameResult, stat *api.FileStat) *api.RenameCardModelFieldResponse {
	out := &api.RenameCardModelFieldResponse{
		Decks:        r.Decks,
		Cards:        int32(r.Cards),
		NotWritten:   nil,
		Error:        api.ErrorCodeUnspecified,
		At:           stat,
		IsUnlevelled: false,
	}
	for _, deck := range r.NotWritten {
		out.NotWritten = append(out.NotWritten, &api.UnwrittenDeck{
			Path: deck.Path, Problem: newWireProblem(deck.Problem),
		})
	}
	return out
}

func newWireTemplate(f format.CardTemplate) *api.CardTemplate {
	return &api.CardTemplate{Name: f.Name, Preamble: f.Preamble, Front: f.Front, Back: f.Back}
}

// NewWireDeck is one deck as the schema carries it. The cards go out in the order
// they stand in the file, and a problem's position is an index into that list.
func NewWireDeck(path, title string, d format.Deck, cutting map[string]string) *api.Deck {
	out := &api.Deck{
		Path:     path,
		Title:    title,
		Preamble: d.Preamble,
		Tail:     d.Tail,
		Sections: make([]*api.Section, 0, len(d.Sections)),
		Cards:    make([]*api.Card, 0, len(d.Cards)),
		Problems: newWireProblems(d.Problems),
	}
	for _, s := range d.Sections {
		out.Sections = append(out.Sections, &api.Section{Name: s.Name, Preamble: s.Preamble})
	}
	for _, card := range d.Cards {
		out.Cards = append(out.Cards, newWireCard(card, cutting[card.CardModelLink]))
	}
	return out
}

func newWireCard(c format.Card, pos string) *api.Card {
	out := &api.Card{
		Heading:       c.Heading,
		CardID:        string(c.CardID),
		SectionIndex:  section(c.Section),
		CardModelLink: c.CardModelLink,
		CardModelPath: pos,
		Preamble:      c.Preamble,
		Values:        make([]*api.Value, 0, len(c.Values)),
	}
	for _, v := range c.Values {
		out.Values = append(out.Values, &api.Value{Field: v.Field, Text: v.Text})
	}
	return out
}

// section maps the section index to an int32 pointer, returning nil for NoSection.
func section(pos int) *int32 {
	if pos == format.NoSection {
		return nil
	}
	index := int32(pos)
	return &index
}

// sectionOf maps a section pointer to domain section index, returning NoSection for nil.
func sectionOf(index *int32) int {
	if index == nil {
		return format.NoSection
	}
	return int(*index)
}

// newWireProblems is what was wrong with a file, in the order it was found. A check
// the schema names no fault for is drawn nowhere, and is left out.
func newWireProblems(problems []format.Problem) []*api.Problem {
	if len(problems) == 0 {
		return nil
	}
	out := make([]*api.Problem, 0, len(problems))
	for _, p := range problems {
		if one := newWireProblem(p); one != nil {
			out = append(out, one)
		}
	}
	return out
}

// newWireProblem is one problem as the schema carries it, and nothing for a fault
// the schema does not name.
func newWireProblem(p format.Problem) *api.Problem {
	kind, isNamed := newWireProblemKind(p.Fault)
	if !isNamed {
		return nil
	}
	return &api.Problem{
		Kind:     kind,
		Card:     position(p.Card),
		Template: position(p.Template),
		Field:    p.Field,
		Text:     p.Detail,
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

// newWireProblemKind returns the kind of a problem as the schema names it, and
// whether the schema names it at all.
func newWireProblemKind(fault format.Fault) (api.ProblemKind, bool) {
	switch fault {
	case format.FaultTwoFields:
		return api.ProblemKindFieldDeclaredTwice, true
	case format.FaultNoFields:
		return api.ProblemKindCardModelWithoutFields, true
	case format.FaultTemplateSide:
		return api.ProblemKindTemplateMissingASide, true
	case format.FaultPlaceholder:
		return api.ProblemKindPlaceholderUndeclared, true
	case format.FaultNoCardModel:
		return api.ProblemKindCardWithoutACardModel, true
	case format.FaultNotACardModel:
		return api.ProblemKindCardModelIsNotOne, true
	case format.FaultDuplicateCardID:
		return api.ProblemKindCardIDCarriedTwice, true
	case format.FaultTwoValues:
		return api.ProblemKindFieldWrittenTwice, true
	case format.FaultNotWritten:
		return api.ProblemKindFieldNotRenamed, true
	case format.FaultTooLarge:
		return api.ProblemKindUnspecified, false
	default:
		return api.ProblemKindUnspecified, false
	}
}

// NewDeck is the deck a client is putting in the vault, in the words the
// core holds one in.
func NewDeck(w *api.WriteDeckRequest) format.Deck {
	out := format.Deck{
		Ref:      domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}},
		Preamble: w.Preamble,
		Tail:     w.Tail,
		Sections: nil,
		Cards:    newCards(w.Cards),
		Problems: nil,
	}
	for _, s := range w.Sections {
		out.Sections = append(out.Sections,
			format.Section{Name: s.Name, Preamble: s.Preamble})
	}
	return out
}

// newCards is the cards a client is putting into a deck, in the words the core
// holds them in.
func newCards(cs []*api.Card) []format.Card {
	if len(cs) == 0 {
		return nil
	}
	out := make([]format.Card, 0, len(cs))
	for _, c := range cs {
		card := format.Card{
			Heading:       c.Heading,
			CardID:        cardid.ID(c.CardID),
			Section:       sectionOf(c.SectionIndex),
			CardModelLink: c.CardModelLink,
			Preamble:      c.Preamble,
			Values:        nil,
		}
		for _, v := range c.Values {
			card.Values = append(card.Values, format.Value{Field: v.Field, Text: v.Text})
		}
		out = append(out, card)
	}
	return out
}

// NewTemplates is the templates a client is putting into a card model.
func NewTemplates(fs []*api.CardTemplate) []format.CardTemplate {
	if len(fs) == 0 {
		return nil
	}
	out := make([]format.CardTemplate, 0, len(fs))
	for _, f := range fs {
		out = append(out, format.CardTemplate{
			Name: f.Name, Preamble: f.Preamble, Front: f.Front, Back: f.Back,
		})
	}
	return out
}
