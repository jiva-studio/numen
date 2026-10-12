package mcp

import (
	"context"
	"errors"
	"fmt"
	"slices"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// How much one call may ask about the cards of a vault. A deck is a file
// holding what would otherwise be a folder of notes, so a deck with no ceiling
// on it is a way to put a vault in a context window by accident.
const (
	maxStencils = 50
	maxCards    = 50
)

// maxDecks is how many decks a vault may hold and still have a field renamed
// through these tools.
//
// A field's name is a heading in every card of every deck its stencil cuts, so
// the rename reads and writes every deck the vault holds, one behind the other
// under the write lock. A vault past this is refused before the stencil is
// written, and the person renames the field in the window.
const maxDecks = 200

// Stencil is a stencil as the list of them names it.
type Stencil struct {
	Path string `json:"path" jsonschema:"the stencil's path relative to the vault folder"`
	// Name is what a card's wikilink writes. A link is resolved by path and by
	// filename, so the name that reaches this stencil is its filename without
	// the extension.
	Name   string   `json:"name" jsonschema:"the name to write in a card's wikilink, which is the filename without its extension"`
	Title  string   `json:"title" jsonschema:"what the stencil is called, which is what a person sees"`
	Fields []string `json:"fields,omitempty" jsonschema:"what a card cut by this stencil is asked for, in order; absent for a stencil that could not be read"`
}

// Card is one card as a deck answers with it.
type Card struct {
	Mark    string       `json:"mark" jsonschema:"what the card is for as long as it exists, and how every tool here addresses it; empty for a card typed in by hand, which is given one the next time the deck is written"`
	Section int          `json:"section" jsonschema:"where the section this card stands under stands in the deck's sections, counted from the first; -1 for a card standing before the first section"`
	Stencil string       `json:"stencil,omitempty" jsonschema:"the stencil this card is cut by, as the wikilink beneath its heading names it"`
	Values  []FieldValue `json:"values,omitempty" jsonschema:"what the card holds, in the order it stands in the file"`
}

// FieldValue is what somebody wrote under one of a card's fields.
type FieldValue struct {
	Field string `json:"field" jsonschema:"the field's name, spelled as the stencil declares it"`
	Text  string `json:"text" jsonschema:"the HTML under that heading"`
}

// Fault is one thing wrong with a deck, on the card it is against.
type Fault struct {
	Card int    `json:"card" jsonschema:"where the card stands in the deck, counted from its first card; -1 for a fault against no card"`
	Why  string `json:"why" jsonschema:"what is wrong, and what the application did about it"`
}

// CurrentCard is the card in front of the person, as an agent is told about it.
// The deck is the file it stands in and the mark is what every card tool
// addresses it by.
type CurrentCard struct {
	Deck string `json:"deck" jsonschema:"the deck the card stands in, by the path the vault files it under; empty when no card is in front of them"`
	Card string `json:"card,omitempty" jsonschema:"the card's mark, as card_read gives it"`
	Face string `json:"face,omitempty" jsonschema:"the face it is being shown through, spelled as the stencil writes it"`
}

func addCardTools(server *sdk.Server, core Core) {
	addCardReadTools(server, core)
	addCardWriteTools(server, core)
}

// WriteOutcome is what a deck became: where it is, and what to present at the
// next write of it.
type WriteOutcome struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint" jsonschema:"hand this to the next write of this deck without reading it back"`
	Cards       int    `json:"cards" jsonschema:"how many cards the deck now holds"`
	Mark        string `json:"mark,omitempty" jsonschema:"the mark the card just written is addressed by, for as long as it exists"`
}

// changeDeck reads a deck, opens its body for splicing, hands it to change, and
// puts back what comes out, with the mark every card that carried none was
// given.
//
// A change is a splice: the run one card, one value or one heading occupies is
// replaced. The preamble, the tail, and every section and card the change did
// not name are written as the bytes they arrived as, down to the blank lines
// and the whitespace the person left at the ends of their lines.
func changeDeck(
	ctx context.Context, core Core, path, fingerprint string,
	change func(read cards.DeckContents, file *format.DeckFile) error,
) (WriteOutcome, []format.CardMark, error) {
	seen, err := parseFingerprint(fingerprint)
	if err != nil {
		return WriteOutcome{}, nil, err
	}
	v := core.getCurrentVault().Vault
	read, err := readDeckIn(ctx, core, v, path)
	if err != nil {
		return WriteOutcome{}, nil, err
	}

	deckEdit := core.Cards.DeckEdit
	if deckEdit == nil {
		deckEdit = format.OpenDeckBody
	}
	file := deckEdit(read.Raw)
	if err := change(read, file); err != nil {
		return WriteOutcome{}, nil, err
	}
	body := file.Body()
	wrote, err := core.Cards.Write.Deck(ctx, v, path, body, seen)
	// A write that reached the vault is a write that happened, so the caller is
	// handed the fingerprint it presents at its next write.
	if err != nil && !errors.Is(err, note.ErrUnlevelled) {
		return WriteOutcome{}, nil, err
	}
	return WriteOutcome{
		Path:        path,
		Fingerprint: fingerprintOf(wrote.Fingerprint),
		Cards:       len(file.Deck(domain.Fingerprint{}).Cards),
	}, wrote.Given, nil
}

// index is where the card of a mark stands. A deck holding no card of it is
// ErrNoSuchCard — a card typed in by hand carries none until the deck is
// written, and no mark reaches it — and a deck holding two is ErrTwoCards:
// both are read and both are shown, and choosing between them would be choosing
// which of the two the person meant.
func index(held []format.Card, carried string) (int, error) {
	at := -1
	if carried != "" {
		for i, card := range held {
			if string(card.Mark) != carried {
				continue
			}
			if at >= 0 {
				return 0, fmt.Errorf("%w: %s", format.ErrTwoCards, carried)
			}
			at = i
		}
	}
	if at < 0 {
		return 0, fmt.Errorf("%w: %s", format.ErrNoSuchCard, carried)
	}
	return at, nil
}

// getMark is the mark the card standing at one place was given, and nothing
// where it carried one already.
func getMark(given []format.CardMark, at int) string {
	for _, one := range given {
		if one.Card == at {
			return string(one.Mark)
		}
	}
	return ""
}

// findSectionEnd is where a card written at the end of one of a deck's sections
// stands, counted from the deck's first card. A card goes at the end of its
// section, which is in front of the first card of a later one.
func findSectionEnd(d format.Deck, section int) (int, error) {
	if section < 0 || section >= len(d.Sections) {
		return 0, fmt.Errorf(
			"this deck has %d sections, so there is none standing at %d", len(d.Sections), section)
	}
	for i, card := range d.Cards {
		if card.Section > section {
			return i, nil
		}
	}
	return len(d.Cards), nil
}

func newCard(card format.Card) Card {
	out := Card{Mark: string(card.Mark), Section: card.Section, Stencil: card.StencilLink}
	for _, v := range card.Values {
		out.Values = append(out.Values, FieldValue{Field: v.Field, Text: v.Text})
	}
	return out
}

// keepFields is the card holding the named fields alone, matched by the name the card
// writes over the value, which is not always the name its stencil declares: a
// deck a rename did not reach writes the old one. A call naming no field asks
// for the whole card.
func keepFields(card Card, fields []string) Card {
	if len(fields) == 0 {
		return card
	}
	kept := card
	kept.Values = nil
	for _, v := range card.Values {
		if slices.Contains(fields, v.Field) {
			kept.Values = append(kept.Values, v)
		}
	}
	return kept
}

func values(vs []FieldValue) []format.Value {
	out := make([]format.Value, 0, len(vs))
	for _, v := range vs {
		out = append(out, format.Value{Field: v.Field, Text: v.Text})
	}
	return out
}

// faults is what was wrong with a deck, on the card each stands against. The
// position is counted from the deck's first card, whichever run of them a call
// asked for.
func faults(problems []format.Problem) []Fault {
	if len(problems) == 0 {
		return nil
	}
	out := make([]Fault, 0, len(problems))
	for _, p := range problems {
		out = append(out, Fault{Card: p.Card, Why: p.Detail})
	}
	return out
}

// countBytes is how many bytes a call will put in a deck. A field's name lands
// in the file beside its value, so both are measured.
func countBytes(vs []FieldValue) int {
	size := 0
	for _, v := range vs {
		size += len(v.Field) + len(v.Text)
	}
	return size
}
