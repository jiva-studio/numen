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
	Text  string `json:"text" jsonschema:"the markdown under that heading"`
}

// Fault is one thing wrong with a deck, on the card it is against.
type Fault struct {
	Card int    `json:"card" jsonschema:"where the card stands in the deck, counted from its first card; -1 for a fault against no card"`
	Why  string `json:"why" jsonschema:"what is wrong, and what the application did about it"`
}

func addCardTools(server *sdk.Server, core Core) {
	addCardReadingTools(server, core)
	addCardWritingTools(server, core)
}

// addCardShowing tells an agent which card the person is looking at.
//
// It is added only where a window says. A binary nobody is sitting at answers
// about a vault and about nothing in front of anybody.
func addCardShowing(server *sdk.Server, core Core) {
	if core.Reviewing == nil {
		return
	}

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_showing",
		Title: "What card the person is looking at",
		Description: "The card in front of the person: the deck it stands in, the mark " +
			"`card_read` addresses it by, and the face it is being shown through. Ask it " +
			"before saying anything about the card they are on — a person answers one " +
			"card and moves to the next while you work — and read the card itself with " +
			"`card_read`. Nothing is in front of them between cards, and the deck comes " +
			"back empty.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, Asked, error) {
		return nil, core.Reviewing(), nil
	})
}

func addCardReadingTools(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_stencil_list",
		Title: "List the stencils a vault holds",
		Description: "The stencils of the vault: where each is filed, the name a card's " +
			"wikilink writes, what it is called, and what a card cut by it is asked for. " +
			"Call this before writing a card, because a card names its stencil and asks " +
			"for the fields that stencil declares. At most 50 come back, and `total` says " +
			"how many the vault holds, so a list shorter than that was cut by the " +
			"ceiling. A stencil with no fields listed is one that could not be read.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Limit int `json:"limit,omitempty" jsonschema:"how many stencils to return, at most 50, which is also the default"`
	}) (*sdk.CallToolResult, struct {
		Stencils []Stencil `json:"stencils"`
		Total    int       `json:"total" jsonschema:"how many stencils the vault holds, which stands above the list when the ceiling was reached"`
	}, error) {
		type out = struct {
			Stencils []Stencil `json:"stencils"`
			Total    int       `json:"total" jsonschema:"how many stencils the vault holds, which stands above the list when the ceiling was reached"`
		}
		if in.Limit > maxStencils {
			return nil, out{}, fmt.Errorf("ask for at most %d stencils at a time", maxStencils)
		}
		limit := maxStencils
		if in.Limit > 0 {
			limit = in.Limit
		}
		held, count, err := core.Cards.List.Execute(ctx, core.shown().Vault, limit)
		if err != nil {
			return nil, out{}, err
		}
		res := out{Total: count}
		for _, s := range held {
			res.Stencils = append(res.Stencils, Stencil{
				Path: s.Path, Name: domain.Basename(s.Path), Title: s.Title, Fields: s.Fields,
			})
		}
		return nil, res, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_read",
		Title: "Read the cards of a deck",
		Description: "The cards of one deck, in the order they stand in the file, each " +
			"under the mark every other tool here addresses it by. A deck holds as many " +
			"cards as a person writes, so this answers a run of them at a time: `total` " +
			"says how many there are, and `from` takes the next run. The sections are " +
			"the runs a person divided the deck into, and every card says which of them " +
			"it stands under. What was wrong with the file comes back under `faults`, " +
			"on the card it is against. The fingerprint is what the writing tools want: " +
			"hand it back and a write is refused if the person changed the deck in the " +
			"meantime.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path   string   `json:"path" jsonschema:"the deck to read"`
		From   int      `json:"from,omitempty" jsonschema:"which card to start at, counted from zero"`
		Limit  int      `json:"limit,omitempty" jsonschema:"how many cards to return"`
		Mark   string   `json:"mark,omitempty" jsonschema:"one card's mark, to read that card alone; from and limit say nothing about a call naming it"`
		Fields []string `json:"fields,omitempty" jsonschema:"the fields to answer with, spelled as the card writes them, which is what card_read gives; every other value is left out, and a card writing none of them comes back holding nothing"`
	}) (*sdk.CallToolResult, struct {
		Cards       []Card   `json:"cards"`
		Sections    []string `json:"sections,omitempty" jsonschema:"what the deck's sections are called, in the order they stand in the file"`
		Total       int      `json:"total" jsonschema:"how many cards the deck holds"`
		Faults      []Fault  `json:"faults,omitempty"`
		Fingerprint string   `json:"fingerprint" jsonschema:"hand this to a writing tool to refuse a write over an edit you did not see"`
	}, error) {
		type out = struct {
			Cards       []Card   `json:"cards"`
			Sections    []string `json:"sections,omitempty" jsonschema:"what the deck's sections are called, in the order they stand in the file"`
			Total       int      `json:"total" jsonschema:"how many cards the deck holds"`
			Faults      []Fault  `json:"faults,omitempty"`
			Fingerprint string   `json:"fingerprint" jsonschema:"hand this to a writing tool to refuse a write over an edit you did not see"`
		}
		// One card is one card however many were asked for, so the ceiling is
		// put on the call that reads a run of them.
		if in.Mark == "" && in.Limit > maxCards {
			return nil, out{}, fmt.Errorf("read at most %d cards at a time", maxCards)
		}
		read, err := core.Cards.Read.Deck(ctx, core.shown().Vault, in.Path)
		if err != nil {
			return nil, out{}, err
		}
		if why := whyNotADeck(read); why != "" {
			return nil, out{}, fmt.Errorf("%s: %s", in.Path, why)
		}

		limit := maxCards
		if in.Limit > 0 {
			limit = in.Limit
		}
		from := min(max(in.From, 0), len(read.Body.Cards))
		res := out{
			Total:       len(read.Body.Cards),
			Faults:      faults(read.Body.Problems),
			Fingerprint: fingerprintOf(read.Fingerprint),
		}
		for _, s := range read.Body.Sections {
			res.Sections = append(res.Sections, s.Name)
		}
		if in.Mark != "" {
			at, err := standing(read.Body.Cards, in.Mark)
			if err != nil {
				return nil, out{}, err
			}
			res.Cards = append(res.Cards, only(carded(read.Body.Cards[at]), in.Fields))
			return nil, res, nil
		}
		for _, card := range read.Body.Cards[from:min(from+limit, len(read.Body.Cards))] {
			res.Cards = append(res.Cards, only(carded(card), in.Fields))
		}
		return nil, res, nil
	})
}

func addCardWritingTools(server *sdk.Server, core Core) {
	addCardEditingTools(server, core)
	addCardMakingTools(server, core)
}

// addCardEditingTools are what changes the cards of a deck that is already
// there. Nothing here makes a deck or a stencil.
func addCardEditingTools(server *sdk.Server, core Core) {
	// One card per call. A call is written out in full before it is made and
	// this one carries what a person wrote, so each is filed as it is finished.
	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_add",
		Title: "Add a card to a deck",
		Description: "Write one card at the end of a deck, or at the end of one of its " +
			"sections. Name the stencil it is cut by, as `card_stencil_list` gives that " +
			"name under `name`, and give a value for every field that stencil declares, " +
			"the first included. Do not write the markdown of a card yourself: a card " +
			"written by hand without the wikilink under its heading is a card with no " +
			"stencil, and nothing says so until somebody opens the deck. The card's " +
			"heading is written from its first field, and the mark it is addressed by " +
			"comes back under `mark`. The fingerprint from `card_read` is required, and a " +
			"write lands only on the deck that fingerprint names.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string       `json:"path" jsonschema:"the deck to write into"`
		Stencil     string       `json:"stencil" jsonschema:"the stencil it is cut by, by the name card_stencil_list gave under name"`
		Values      []FieldValue `json:"values" jsonschema:"what the card holds, in the order to write it"`
		Section     *int         `json:"section,omitempty" jsonschema:"which of the deck's sections to write it at the end of, counted from the first; left out, the card goes at the end of the deck"`
		Fingerprint string       `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
	}) (*sdk.CallToolResult, WriteOutcome, error) {
		if size := carries(in.Values); size > maxBytes {
			return nil, WriteOutcome{}, fmt.Errorf(
				"a card of %d bytes is more than this writes at once, which is %d", size, maxBytes)
		}
		// Where the card went, so that the mark minted for it is the one this
		// answers with.
		stands := 0
		written, minted, err := changing(ctx, core, in.Path, in.Fingerprint,
			func(read cards.DeckContents, file *format.DeckFile) error {
				card := format.Card{Stencil: in.Stencil, Values: values(in.Values)}
				if in.Section == nil {
					stands = len(read.Body.Cards)
					return file.AddCard(card)
				}
				at, err := placed(read.Body, *in.Section)
				if err != nil {
					return err
				}
				stands = at
				return file.AddCardUnder(*in.Section, card)
			})
		if err != nil {
			return nil, WriteOutcome{}, err
		}
		written.Mark = markOf(minted, stands)
		return nil, written, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_edit",
		Title: "Change what a card holds",
		Description: "Write values into one card of a deck. A field the card already " +
			"carries is replaced, one it does not is added at the end of it, and a field " +
			"left out of the call is left as it stands. Send the fields you are changing " +
			"and no others; taking a field off a card is `card_value_remove`. " +
			"The card is addressed by the " +
			"mark `card_read` gives it, which is what the card is for as long as it " +
			"exists: a card rewritten from end to end is still that card, and what is " +
			"attached to it stays attached. The first field is written like every " +
			"other, and the card's heading follows it. The fingerprint from `card_read` is " +
			"required, and a write lands only on the deck that fingerprint names.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string       `json:"path" jsonschema:"the deck the card is in"`
		Mark        string       `json:"mark" jsonschema:"the card's mark, as card_read gives it"`
		Values      []FieldValue `json:"values" jsonschema:"the fields to write, and what to put under each"`
		Stencil     string       `json:"stencil,omitempty" jsonschema:"the stencil it is cut by from now on, by the name card_stencil_list gave under name; left out, the card keeps the one it names"`
		Fingerprint string       `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
	}) (*sdk.CallToolResult, WriteOutcome, error) {
		if size := carries(in.Values); size > maxBytes {
			return nil, WriteOutcome{}, fmt.Errorf(
				"a card of %d bytes is more than this writes at once, which is %d", size, maxBytes)
		}
		written, _, err := changing(ctx, core, in.Path, in.Fingerprint,
			func(read cards.DeckContents, file *format.DeckFile) error {
				card := domain.CardID(in.Mark)
				// A call writing nothing still says whether the deck holds the
				// card it was addressed to.
				if _, err := standing(read.Body.Cards, in.Mark); err != nil {
					return err
				}
				if in.Stencil != "" {
					if err := file.SetStencil(card, in.Stencil); err != nil {
						return err
					}
				}
				for _, v := range in.Values {
					if err := file.SetValue(card, v.Field, v.Text); err != nil {
						return err
					}
				}
				return nil
			})
		return nil, written, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_value_remove",
		Title: "Take a field off a card",
		Description: "Remove one field from one card: its heading, and what the person " +
			"wrote under it. `card_edit` writes a value and never takes one away, and an " +
			"empty value is a field standing empty, not a field gone. The fields around " +
			"it keep the order they were written in, and every other byte of the file is " +
			"left as it was. The stencil is not touched: it goes on declaring the field, " +
			"and every other card cut by it goes on carrying its own value. A card " +
			"writing nothing under that field is refused. The card is addressed by the " +
			"mark `card_read` gives it, the fingerprint from `card_read` is required, and " +
			"a write lands only on the deck that fingerprint names.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string `json:"path" jsonschema:"the deck the card is in"`
		Mark        string `json:"mark" jsonschema:"the card's mark, as card_read gives it"`
		Field       string `json:"field" jsonschema:"the field to take off, as card_read gives it under field"`
		Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
	}) (*sdk.CallToolResult, WriteOutcome, error) {
		written, _, err := changing(ctx, core, in.Path, in.Fingerprint,
			func(_ cards.DeckContents, file *format.DeckFile) error {
				return file.RemoveValue(domain.CardID(in.Mark), in.Field)
			})
		return nil, written, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_remove",
		Title: "Take a card out of a deck",
		Description: "Remove one card from a deck: its heading, the stencil it named and " +
			"every value under it. The rest of the file is left the bytes it was, down to " +
			"the blank lines and the whitespace at the ends of the lines; what every " +
			"write of a deck puts right is the mark of a card carrying none and the " +
			"heading of a card that has fallen out of step with its first field. " +
			"Nothing brings the card back, so read the deck before removing from it. The " +
			"fingerprint from `card_read` is required, and a write lands only on the deck " +
			"that fingerprint names.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string `json:"path" jsonschema:"the deck the card is in"`
		Mark        string `json:"mark" jsonschema:"the card's mark, as card_read gives it"`
		Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
	}) (*sdk.CallToolResult, WriteOutcome, error) {
		written, _, err := changing(ctx, core, in.Path, in.Fingerprint,
			func(_ cards.DeckContents, file *format.DeckFile) error {
				return file.RemoveCard(domain.CardID(in.Mark))
			})
		return nil, written, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_section_add",
		Title: "Divide a deck with a section",
		Description: "Write a section at the end of a deck. A section is a name a person " +
			"gives one run of a deck, and it is a name and nothing else: it carries no " +
			"field, no stencil and no schedule. `card_add` writes a card at the end of " +
			"one, and `card_read` says which section each card stands under. The " +
			"fingerprint from `card_read` is required, and a write lands only on the deck " +
			"that fingerprint names.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string `json:"path" jsonschema:"the deck to divide"`
		Name        string `json:"name" jsonschema:"what the section is called"`
		Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
	}) (*sdk.CallToolResult, WriteOutcome, error) {
		written, _, err := changing(ctx, core, in.Path, in.Fingerprint,
			func(_ cards.DeckContents, file *format.DeckFile) error {
				return file.AddSection(format.Section{Name: in.Name})
			})
		return nil, written, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_section_rename",
		Title: "Rename a section of a deck",
		Description: "Give one of a deck's sections a different name. Only the heading " +
			"line is written: the cards standing under it stay where they are, keep their " +
			"marks and keep what they hold, and what a person wrote beneath the heading is " +
			"left as it stands. The section is named by where it stands in the deck, " +
			"counted from the first, which is what `card_read` gives under `sections`. The " +
			"fingerprint from `card_read` is required, and a write lands only on the deck " +
			"that fingerprint names.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string `json:"path" jsonschema:"the deck the section is in"`
		Section     int    `json:"section" jsonschema:"where the section stands in the deck's sections, counted from the first, as card_read gives them"`
		Name        string `json:"name" jsonschema:"what it is called from now on"`
		Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
	}) (*sdk.CallToolResult, WriteOutcome, error) {
		written, _, err := changing(ctx, core, in.Path, in.Fingerprint,
			func(_ cards.DeckContents, file *format.DeckFile) error {
				return file.RenameSection(in.Section, in.Name)
			})
		return nil, written, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_section_remove",
		Title: "Take a section out of a deck",
		Description: "Remove the heading of one of a deck's sections. A section is a name " +
			"and nothing else, so this takes away the name alone: no card is removed, and " +
			"the cards that stood under the heading stay in the file, in the order they " +
			"were, under whatever section now stands above them. What a person wrote " +
			"beneath the heading stays there too. The section is named by where it stands " +
			"in the deck, counted from the first, which is what `card_read` gives under " +
			"`sections`. The fingerprint from `card_read` is required, and a write lands " +
			"only on the deck that fingerprint names.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string `json:"path" jsonschema:"the deck the section is in"`
		Section     int    `json:"section" jsonschema:"where the section stands in the deck's sections, counted from the first, as card_read gives them"`
		Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
	}) (*sdk.CallToolResult, WriteOutcome, error) {
		written, _, err := changing(ctx, core, in.Path, in.Fingerprint,
			func(_ cards.DeckContents, file *format.DeckFile) error {
				return file.RemoveSection(in.Section)
			})
		return nil, written, err
	})
}

// addCardMakingTools are what a vault is arranged into: a deck, a stencil, and
// the name a stencil gives a field wherever it is written.
func addCardMakingTools(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_deck_create",
		Title: "Create a deck",
		Description: "Make a deck of no cards, and fill it with `card_add`. A deck is a " +
			"file of cards and says so from the moment it exists, so it is a deck to " +
			"everything that reads the vault before a card is written into it. Make a " +
			"deck here: a note made with `note_create` is an ordinary note, and " +
			"`card_add` writes into a deck alone. The deck is named after its title.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Title  string `json:"title" jsonschema:"what the deck is called"`
		Folder string `json:"folder,omitempty" jsonschema:"where to file it, relative to the vault folder; the root by default"`
	}) (*sdk.CallToolResult, cards.CreateNoteResult, error) {
		made, err := core.Cards.Create.Deck(ctx, core.shown().Vault, cards.New{
			Title: in.Title, Folder: in.Folder,
		})
		return nil, made, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_stencil_create",
		Title: "Create a stencil",
		Description: "Make a stencil: the fields a card is asked for, in the order to ask " +
			"for them, and the faces one is shown through. A face is markdown with " +
			"`{{Field}}` standing where a value goes, and every name in braces must be " +
			"one of the fields. A card's heading is read from the first field, so there " +
			"is at least one; it holds as many lines as a person writes, and the " +
			"heading is its first line. The stencil is filed under its title, and the " +
			"name a card's wikilink writes is that filename, which `card_stencil_list` " +
			"gives under `name`.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Title  string   `json:"title" jsonschema:"what the stencil is called"`
		Fields []string `json:"fields" jsonschema:"the names of the fields, in the order a person is asked for them; a card's heading is read from the first"`
		Faces  []Face   `json:"faces" jsonschema:"the ways a card cut by this stencil is shown"`
		Folder string   `json:"folder,omitempty" jsonschema:"where to file it, relative to the vault folder; the root by default"`
	}) (*sdk.CallToolResult, cards.CreateNoteResult, error) {
		body, err := core.Cards.StencilBody("", faced(in.Faces), "")
		if err != nil {
			return nil, cards.CreateNoteResult{}, err
		}
		if len(body) > maxBytes {
			return nil, cards.CreateNoteResult{}, fmt.Errorf(
				"a stencil of %d bytes is more than this writes at once, which is %d", len(body), maxBytes)
		}
		made, err := core.Cards.Create.Stencil(ctx, core.shown().Vault, cards.New{
			Title: in.Title, Body: body, Folder: in.Folder, Fields: in.Fields,
		})
		return nil, made, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_field_rename",
		Title: "Rename a stencil's field",
		Description: "Give one of a stencil's fields a different name, everywhere it is " +
			"written. A field's name stands in the stencil that declares it and as a " +
			"heading in every card of every deck that stencil cuts, so renaming it in " +
			"one place alone leaves values under a heading nothing declares. This " +
			"rewrites them all and leaves what stands under each heading as it was. " +
			"Every field a stencil declares stands under its own heading in every card, " +
			"the first included, so renaming any of them reaches the decks. A " +
			"deck it could not be written to comes back under `notWritten` and keeps the " +
			"old heading.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path string `json:"path" jsonschema:"the stencil that declares the field"`
		From string `json:"from" jsonschema:"the field's name now"`
		To   string `json:"to" jsonschema:"what it is called from now on"`
	}) (*sdk.CallToolResult, struct {
		Decks      []string `json:"decks,omitempty" jsonschema:"the decks a heading was rewritten in"`
		Cards      int      `json:"cards" jsonschema:"how many headings were rewritten"`
		NotWritten []string `json:"notWritten,omitempty" jsonschema:"the decks the rename could not be written to, which keep the old heading"`
	}, error) {
		type out = struct {
			Decks      []string `json:"decks,omitempty" jsonschema:"the decks a heading was rewritten in"`
			Cards      int      `json:"cards" jsonschema:"how many headings were rewritten"`
			NotWritten []string `json:"notWritten,omitempty" jsonschema:"the decks the rename could not be written to, which keep the old heading"`
		}
		renamed, err := core.Cards.RenameField.Execute(ctx, core.shown().Vault, cards.Rename{
			Stencil: in.Path, From: in.From, To: in.To,
		})
		if err != nil {
			return nil, out{}, err
		}
		res := out{Decks: renamed.Decks, Cards: renamed.Cards}
		for _, deck := range renamed.NotWritten {
			res.NotWritten = append(res.NotWritten, deck.Path)
		}
		return nil, res, nil
	})
}

// Face is one way a stencil shows a card.
type Face struct {
	Name  string `json:"name" jsonschema:"what the face is called"`
	Front string `json:"front" jsonschema:"what is shown before the answer, as markdown with {{Field}} where a value goes"`
	Back  string `json:"back" jsonschema:"what is shown after it"`
}

// WriteOutcome is what a deck became: where it is, and what to present at the
// next write of it.
type WriteOutcome struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint" jsonschema:"hand this to the next write of this deck without reading it back"`
	Cards       int    `json:"cards" jsonschema:"how many cards the deck now holds"`
	Mark        string `json:"mark,omitempty" jsonschema:"the mark the card just written is addressed by, for as long as it exists"`
}

// changing reads a deck, opens its body for splicing, hands it to change, and
// puts back what comes out, with the mark every card that carried none was
// given.
//
// A change is a splice: the run one card, one value or one heading occupies is
// replaced. The preamble, the tail, and every section and card the change did
// not name are written as the bytes they arrived as, down to the blank lines
// and the whitespace the person left at the ends of their lines.
func changing(
	ctx context.Context, core Core, path, fingerprint string,
	change func(read cards.DeckContents, file *format.DeckFile) error,
) (WriteOutcome, []format.Minted, error) {
	seen, err := parseFingerprint(fingerprint)
	if err != nil {
		return WriteOutcome{}, nil, err
	}
	v := core.shown().Vault
	read, err := core.Cards.Read.Deck(ctx, v, path)
	if err != nil {
		return WriteOutcome{}, nil, err
	}
	if why := whyNotADeck(read); why != "" {
		return WriteOutcome{}, nil, fmt.Errorf("%s: %s", path, why)
	}

	file := core.Cards.DeckEdit(read.Raw)
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
	}, wrote.Minted, nil
}

// standing is where the card of a mark stands. A deck holding no card of it is
// ErrNoSuchCard — a card typed in by hand carries none until the deck is
// written, and no mark reaches it — and a deck holding two is ErrTwoCards:
// both are read and both are shown, and choosing between them would be choosing
// which of the two the person meant.
func standing(held []format.Card, carried string) (int, error) {
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

// markOf is the mark the card standing at one place was given, and nothing
// where it carried one already.
func markOf(minted []format.Minted, at int) string {
	for _, one := range minted {
		if one.Card == at {
			return string(one.Mark)
		}
	}
	return ""
}

// placed is where a card written at the end of one of a deck's sections stands,
// counted from the deck's first card. A card goes at the end of its section,
// which is in front of the first card of a later one.
func placed(d format.Deck, section int) (int, error) {
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

// whyNotADeck says why a path is no deck to write cards into, and nothing where
// it is one.
func whyNotADeck(read cards.DeckContents) string {
	if read.Outcome == note.Ok && read.Type != domain.TypeDeck {
		return "this note is not a deck"
	}
	switch read.Outcome {
	case note.Ok:
		return ""
	case note.TooLarge:
		return fmt.Sprintf("it is %d bytes, larger than the %d a deck is read at; open the file instead",
			read.Fingerprint.Size, cards.MaxBytes)
	case note.Missing:
		return "there is no note at this path"
	case note.NotANote:
		return "this is not a note the vault holds"
	case note.NotText:
		return "this file is not text: some of it is not valid UTF-8, so open it as a file"
	case note.Unreadable:
		return "the frontmatter of this note cannot be read, so it can be neither read nor written from here"
	}
	return string(read.Outcome)
}

func carded(card format.Card) Card {
	out := Card{Mark: string(card.Mark), Section: card.Section, Stencil: card.Stencil}
	for _, v := range card.Values {
		out.Values = append(out.Values, FieldValue{Field: v.Field, Text: v.Text})
	}
	return out
}

// only is the card holding the named fields alone, matched by the name the card
// writes over the value, which is not always the name its stencil declares: a
// deck a rename did not reach writes the old one. A call naming no field asks
// for the whole card.
func only(card Card, fields []string) Card {
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

func faced(fs []Face) []format.FaceTemplate {
	out := make([]format.FaceTemplate, 0, len(fs))
	for _, f := range fs {
		out = append(out, format.FaceTemplate{Name: f.Name, Front: f.Front, Back: f.Back})
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

// carries is how many bytes a call will put in a deck. A field's name lands in
// the file beside its value, so both are measured.
func carries(vs []FieldValue) int {
	size := 0
	for _, v := range vs {
		size += len(v.Field) + len(v.Text)
	}
	return size
}
