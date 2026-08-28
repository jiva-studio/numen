package mcp

import (
	"context"
	"fmt"
	"slices"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	format "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cards"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
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
	Mark    string  `json:"mark" jsonschema:"what the card is for as long as it exists, and how every tool here addresses it; empty for a card typed in by hand, which is given one the next time the deck is written"`
	Section int     `json:"section" jsonschema:"where the section this card stands under stands in the deck's sections, counted from the first; -1 for a card standing before the first section"`
	Stencil string  `json:"stencil,omitempty" jsonschema:"the stencil this card is cut by, as the wikilink beneath its heading names it"`
	Values  []Value `json:"values,omitempty" jsonschema:"what the card holds, in the order it stands in the file"`
}

// Value is what somebody wrote under one of a card's fields.
type Value struct {
	Field string `json:"field" jsonschema:"the field's name, spelled as the stencil declares it"`
	Text  string `json:"text" jsonschema:"the markdown under that heading"`
}

// Fault is one thing wrong with a deck, on the card it is against.
type Fault struct {
	Card int    `json:"card" jsonschema:"where the card stands in the deck, counted from its first card; -1 for a fault against no card"`
	Why  string `json:"why" jsonschema:"what is wrong, and what the application did about it"`
}

func addCardTools(server *sdk.Server, core Core) {
	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_stencils",
		Title: "List the stencils a vault holds",
		Description: "The stencils of the vault: where each is filed, the name a card's " +
			"wikilink writes, what it is called, and what a card cut by it is asked for. " +
			"Call this before writing a card, because a card names its stencil and asks " +
			"for the fields that stencil declares. At most 50 come back, and `held` says " +
			"how many the vault holds, so a list shorter than that was cut by the " +
			"ceiling. A stencil with no fields listed is one that could not be read.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Limit int `json:"limit,omitempty" jsonschema:"how many stencils to return, at most 50, which is also the default"`
	}) (*sdk.CallToolResult, struct {
		Stencils []Stencil `json:"stencils"`
		Held     int       `json:"held" jsonschema:"how many stencils the vault holds, which stands above the list when the ceiling was reached"`
	}, error) {
		type out = struct {
			Stencils []Stencil `json:"stencils"`
			Held     int       `json:"held" jsonschema:"how many stencils the vault holds, which stands above the list when the ceiling was reached"`
		}
		if in.Limit > maxStencils {
			return nil, out{}, fmt.Errorf("ask for at most %d stencils at a time", maxStencils)
		}
		limit := maxStencils
		if in.Limit > 0 {
			limit = in.Limit
		}
		held, count, err := core.Stencils.Execute(ctx, core.shown().Vault, limit)
		if err != nil {
			return nil, out{}, err
		}
		res := out{Held: count}
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
			"cards as a person writes, so this answers a run of them at a time: `held` " +
			"says how many there are, and `from` takes the next run. The sections are " +
			"the runs a person divided the deck into, and every card says which of them " +
			"it stands under. What was wrong with the file comes back under `faults`, " +
			"on the card it is against. The fingerprint is what the writing tools want: " +
			"hand it back and a write is refused if the person changed the deck in the " +
			"meantime.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path  string `json:"path" jsonschema:"the deck to read"`
		From  int    `json:"from,omitempty" jsonschema:"which card to start at, counted from zero"`
		Limit int    `json:"limit,omitempty" jsonschema:"how many cards to return"`
	}) (*sdk.CallToolResult, struct {
		Cards       []Card   `json:"cards"`
		Sections    []string `json:"sections,omitempty" jsonschema:"what the deck's sections are called, in the order they stand in the file"`
		Held        int      `json:"held" jsonschema:"how many cards the deck holds"`
		Faults      []Fault  `json:"faults,omitempty"`
		Fingerprint string   `json:"fingerprint" jsonschema:"hand this to a writing tool to refuse a write over an edit you did not see"`
	}, error) {
		type out = struct {
			Cards       []Card   `json:"cards"`
			Sections    []string `json:"sections,omitempty" jsonschema:"what the deck's sections are called, in the order they stand in the file"`
			Held        int      `json:"held" jsonschema:"how many cards the deck holds"`
			Faults      []Fault  `json:"faults,omitempty"`
			Fingerprint string   `json:"fingerprint" jsonschema:"hand this to a writing tool to refuse a write over an edit you did not see"`
		}
		if in.Limit > maxCards {
			return nil, out{}, fmt.Errorf("read at most %d cards at a time", maxCards)
		}
		read, err := core.Cards.Deck(ctx, core.shown().Vault, in.Path)
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
		from := min(max(in.From, 0), len(read.Deck.Cards))
		res := out{
			Held:        len(read.Deck.Cards),
			Faults:      faults(read.Deck.Problems),
			Fingerprint: fingerprintOf(read.Ref),
		}
		for _, s := range read.Deck.Sections {
			res.Sections = append(res.Sections, s.Name)
		}
		for _, card := range read.Deck.Cards[from:min(from+limit, len(read.Deck.Cards))] {
			res.Cards = append(res.Cards, carded(card))
		}
		return nil, res, nil
	})

	// One card per call. A call is written out in full before it is made and
	// this one carries what a person wrote, so each is filed as it is finished.
	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_add",
		Title: "Add a card to a deck",
		Description: "Write one card at the end of a deck, or at the end of one of its " +
			"sections. Name the stencil it is cut by, as `card_stencils` gives that " +
			"name under `name`, and give a value for every field that stencil declares, " +
			"the first included. Do not write the markdown of a card yourself: a card " +
			"written by hand without the wikilink under its heading is a card with no " +
			"stencil, and nothing says so until somebody opens the deck. The card's " +
			"heading is written from its first field, and the mark it is addressed by " +
			"is minted where the deck is written, so `card_read` is what says which " +
			"card this became.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string  `json:"path" jsonschema:"the deck to write into"`
		Stencil     string  `json:"stencil" jsonschema:"the stencil it is cut by, by the name card_stencils gave under name"`
		Values      []Value `json:"values" jsonschema:"what the card holds, in the order to write it"`
		Section     *int    `json:"section,omitempty" jsonschema:"which of the deck's sections to write it at the end of, counted from the first; left out, the card goes at the end of the deck"`
		Fingerprint string  `json:"fingerprint,omitempty" jsonschema:"what card_read said the deck was, to refuse a write over somebody else's edit"`
	}) (*sdk.CallToolResult, Written, error) {
		if size := carries(in.Values); size > maxBytes {
			return nil, Written{}, fmt.Errorf(
				"a card of %d bytes is more than this writes at once, which is %d", size, maxBytes)
		}
		written, err := changing(ctx, core, in.Path, in.Fingerprint,
			func(read cards.Deck) (format.Deck, error) {
				held := read.Deck
				at, under, err := placed(held, in.Section)
				if err != nil {
					return format.Deck{}, err
				}
				card := format.Card{
					Stencil: in.Stencil, Section: under, Values: values(in.Values),
				}
				held.Cards = slices.Insert(held.Cards, at, card)
				return held, nil
			})
		return nil, written, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_edit",
		Title: "Change what a card holds",
		Description: "Write values into one card of a deck. A field the card already " +
			"carries is replaced, one it does not is added at the end of it, and a field " +
			"left out of the call is left as it stands. The card is addressed by the " +
			"mark `card_read` gives it, which is what the card is for as long as it " +
			"exists: a card rewritten from end to end is still that card, and what is " +
			"attached to it stays attached. The first field is written like every " +
			"other, and the card's heading follows it.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string  `json:"path" jsonschema:"the deck the card is in"`
		Card        string  `json:"card" jsonschema:"the card's mark, as card_read gives it"`
		Values      []Value `json:"values" jsonschema:"the fields to write, and what to put under each"`
		Stencil     string  `json:"stencil,omitempty" jsonschema:"the stencil it is cut by from now on, by the name card_stencils gave under name; left out, the card keeps the one it names"`
		Fingerprint string  `json:"fingerprint,omitempty" jsonschema:"what card_read said the deck was, to refuse a write over somebody else's edit"`
	}) (*sdk.CallToolResult, Written, error) {
		if size := carries(in.Values); size > maxBytes {
			return nil, Written{}, fmt.Errorf(
				"a card of %d bytes is more than this writes at once, which is %d", size, maxBytes)
		}
		written, err := changing(ctx, core, in.Path, in.Fingerprint,
			func(read cards.Deck) (format.Deck, error) {
				held := read.Deck
				at := standing(held.Cards, in.Card)
				if at < 0 {
					return format.Deck{}, fmt.Errorf("%w: %s", format.ErrNoSuchCard, in.Card)
				}
				if in.Stencil != "" {
					held.Cards[at].Stencil = in.Stencil
				}
				for _, v := range in.Values {
					held.Cards[at] = filled(held.Cards[at], v)
				}
				return held, nil
			})
		return nil, written, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_remove",
		Title: "Take a card out of a deck",
		Description: "Remove one card from a deck: its heading, the stencil it named and " +
			"every value under it. The rest of the file is left the bytes it was. " +
			"Nothing brings it back, so read the deck before removing from it.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string `json:"path" jsonschema:"the deck the card is in"`
		Card        string `json:"card" jsonschema:"the card's mark, as card_read gives it"`
		Fingerprint string `json:"fingerprint,omitempty" jsonschema:"what card_read said the deck was, to refuse a write over somebody else's edit"`
	}) (*sdk.CallToolResult, Written, error) {
		written, err := changing(ctx, core, in.Path, in.Fingerprint,
			func(read cards.Deck) (format.Deck, error) {
				held := read.Deck
				at := standing(held.Cards, in.Card)
				if at < 0 {
					return format.Deck{}, fmt.Errorf("%w: %s", format.ErrNoSuchCard, in.Card)
				}
				held.Cards = slices.Delete(held.Cards, at, at+1)
				return held, nil
			})
		return nil, written, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_section_add",
		Title: "Divide a deck with a section",
		Description: "Write a section at the end of a deck. A section is a name a person " +
			"gives one run of a deck, and it is a name and nothing else: it carries no " +
			"field, no stencil and no schedule. `card_add` writes a card at the end of " +
			"one, and `card_read` says which section each card stands under.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Path        string `json:"path" jsonschema:"the deck to divide"`
		Name        string `json:"name" jsonschema:"what the section is called"`
		Fingerprint string `json:"fingerprint,omitempty" jsonschema:"what card_read said the deck was, to refuse a write over somebody else's edit"`
	}) (*sdk.CallToolResult, Written, error) {
		written, err := changing(ctx, core, in.Path, in.Fingerprint,
			func(read cards.Deck) (format.Deck, error) {
				held := read.Deck
				held.Sections = append(held.Sections, format.Section{Name: in.Name})
				return held, nil
			})
		return nil, written, err
	})

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
	}) (*sdk.CallToolResult, cards.Made, error) {
		made, err := core.Cutting.Deck(ctx, core.shown().Vault, cards.New{
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
			"name a card's wikilink writes is that filename, which `card_stencils` " +
			"gives under `name`.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in struct {
		Title  string   `json:"title" jsonschema:"what the stencil is called"`
		Fields []string `json:"fields" jsonschema:"the names of the fields, in the order a person is asked for them; a card's heading is read from the first"`
		Faces  []Face   `json:"faces" jsonschema:"the ways a card cut by this stencil is shown"`
		Folder string   `json:"folder,omitempty" jsonschema:"where to file it, relative to the vault folder; the root by default"`
	}) (*sdk.CallToolResult, cards.Made, error) {
		body, err := core.StencilBody("", faced(in.Faces), "")
		if err != nil {
			return nil, cards.Made{}, err
		}
		if len(body) > maxBytes {
			return nil, cards.Made{}, fmt.Errorf(
				"a stencil of %d bytes is more than this writes at once, which is %d", len(body), maxBytes)
		}
		made, err := core.Cutting.Stencil(ctx, core.shown().Vault, cards.New{
			Title: in.Title, Body: body, Folder: in.Folder, Fields: in.Fields,
		})
		return nil, made, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:  "card_rename_field",
		Title: "Rename a field of a stencil",
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
		renamed, err := core.FieldRename.Execute(ctx, core.shown().Vault, cards.Field{
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

// Written is what a deck became: where it is, and what to present at the next
// write of it.
type Written struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint" jsonschema:"hand this to the next write of this deck without reading it back"`
	Cards       int    `json:"cards" jsonschema:"how many cards the deck now holds"`
}

// changing reads a deck, hands it to change, and puts back what comes out. The
// preamble, the tail, every section and every card change did not touch are
// written as the bytes they arrived as.
func changing(
	ctx context.Context, core Core, path, fingerprint string,
	change func(cards.Deck) (format.Deck, error),
) (Written, error) {
	seen, err := parseFingerprint(fingerprint)
	if err != nil {
		return Written{}, err
	}
	v := core.shown().Vault
	read, err := core.Cards.Deck(ctx, v, path)
	if err != nil {
		return Written{}, err
	}
	if why := whyNotADeck(read); why != "" {
		return Written{}, fmt.Errorf("%s: %s", path, why)
	}

	held, err := change(read)
	if err != nil {
		return Written{}, err
	}
	body, err := core.DeckBody(held)
	if err != nil {
		return Written{}, err
	}
	// The deck this read came out of is what the write lands on where the
	// caller presented nothing of its own.
	if seen == (domain.FileRef{}) {
		seen = read.Ref
	}
	at, err := core.Cuts.Deck(ctx, v, path, body, seen)
	if err != nil {
		return Written{}, err
	}
	return Written{Path: path, Fingerprint: fingerprintOf(at), Cards: len(held.Cards)}, nil
}

// standing is where the card of a mark stands, and -1 where the deck holds
// none. A card typed in by hand carries no mark until the deck is written, and
// no mark reaches it.
func standing(held []format.Card, carried string) int {
	if carried == "" {
		return -1
	}
	for at, card := range held {
		if card.Mark == carried {
			return at
		}
	}
	return -1
}

// placed is where a card being written goes in a deck, and which section it
// stands under. A call naming no section writes at the end of the deck, under
// whatever section stands last.
func placed(d format.Deck, section *int) (at, under int, err error) {
	if section == nil {
		return len(d.Cards), len(d.Sections) - 1, nil
	}
	if *section < 0 || *section >= len(d.Sections) {
		return 0, 0, fmt.Errorf(
			"this deck has %d sections, so there is none standing at %d", len(d.Sections), *section)
	}
	for i, card := range d.Cards {
		if card.Section > *section {
			return i, *section, nil
		}
	}
	return len(d.Cards), *section, nil
}

// holding is the card with one value written into it. A field the card carries
// is replaced where it stands, and one it does not is added at the end.
func filled(card format.Card, v Value) format.Card {
	for at, held := range card.Values {
		if held.Field == v.Field {
			card.Values[at].Text = v.Text
			return card
		}
	}
	card.Values = append(card.Values, format.Value{Field: v.Field, Text: v.Text})
	return card
}

// whyNotADeck says why a path is no deck to write cards into, and nothing where
// it is one.
func whyNotADeck(read cards.Deck) string {
	if read.Outcome == note.Ok && read.Type != domain.TypeDeck {
		return "this note is not a deck"
	}
	switch read.Outcome {
	case note.Ok:
		return ""
	case note.TooLarge:
		return fmt.Sprintf("it is %d bytes, larger than the %d a deck is read at; open the file instead",
			read.Ref.Size, cards.MaxBytes)
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
	out := Card{Mark: card.Mark, Section: card.Section, Stencil: card.Stencil}
	for _, v := range card.Values {
		out.Values = append(out.Values, Value{Field: v.Field, Text: v.Text})
	}
	return out
}

func values(vs []Value) []format.Value {
	out := make([]format.Value, 0, len(vs))
	for _, v := range vs {
		out = append(out, format.Value{Field: v.Field, Text: v.Text})
	}
	return out
}

func faced(fs []Face) []format.Face {
	out := make([]format.Face, 0, len(fs))
	for _, f := range fs {
		out = append(out, format.Face{Name: f.Name, Front: f.Front, Back: f.Back})
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
func carries(vs []Value) int {
	size := 0
	for _, v := range vs {
		size += len(v.Field) + len(v.Text)
	}
	return size
}
