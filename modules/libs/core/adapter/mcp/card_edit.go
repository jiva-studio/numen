package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
)

// addCardEditTools adds the tools that change the cards of an existing deck.
func addCardEditTools(server *sdk.Server, core Core) {
	addCardAdd(server, core)
	addCardEdit(server, core)
	addCardValueRemove(server, core)
	addCardRemove(server, core)
	addCardSectionAdd(server, core)
	addCardSectionRename(server, core)
	addCardSectionRemove(server, core)
}

// checkCardSize returns an error when the card values exceed the size limit of one write.
func checkCardSize(vs []FieldValue) error {
	if size := countBytes(vs); size > maxBytes {
		return fmt.Errorf("a card of %d bytes is more than this writes at once, which is %d", size, maxBytes)
	}
	return nil
}

type cardAddInput struct {
	Path        string       `json:"path" jsonschema:"the deck to write into"`
	Stencil     string       `json:"stencil" jsonschema:"the stencil it is cut by, by the name card_stencil_list gave under name"`
	Values      []FieldValue `json:"values" jsonschema:"what the card holds, in the order to write it"`
	Section     *int         `json:"section,omitempty" jsonschema:"which of the deck's sections to write it at the end of, counted from the first; left out, the card goes at the end of the deck"`
	Fingerprint string       `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

// addCardAdd adds the tool that writes one card per call.
func addCardAdd(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "card_add",
		Title: "Add a card",
		Description: "Write one card at the end of a deck, or at the end of one of its " +
			"sections. Name the stencil it is cut by, as `card_stencil_list` gives that " +
			"name under `name`, and give a value for every field that stencil declares, " +
			"the first included. Do not write a card into the deck yourself: a card " +
			"written by hand without the wikilink under its heading is a card with no " +
			"stencil, and nothing says so until somebody opens the deck. The card's " +
			"heading is written from its first field, and the mark it is addressed by " +
			"comes back under `mark`. The fingerprint from `card_read` is required, and a " +
			"write lands only on the deck that fingerprint names.",
	}, newHandler(func(ctx context.Context, in cardAddInput) (WriteOutcome, error) {
		return writeCardAdd(ctx, core, in)
	}))
}

func writeCardAdd(ctx context.Context, core Core, in cardAddInput) (WriteOutcome, error) {
	if err := checkCardSize(in.Values); err != nil {
		return WriteOutcome{}, err
	}
	// position is the index of the new card in the deck.
	position := 0
	written, given, err := changeDeck(ctx, core, in.Path, in.Fingerprint,
		func(contents cards.DeckContents, file *format.DeckFile) error {
			at, err := insertCard(contents, file, in)
			position = at
			return err
		})
	if err != nil {
		return WriteOutcome{}, err
	}
	written.Mark = getMark(given, position)
	return written, nil
}

// insertCard writes the card at the end of the deck, or at the end of the
// section the input names, and returns its index.
func insertCard(contents cards.DeckContents, file *format.DeckFile, in cardAddInput) (int, error) {
	card := format.Card{StencilLink: in.Stencil, Values: values(in.Values)}
	if in.Section == nil {
		return len(contents.Body.Cards), file.AddCard(card)
	}
	at, err := findSectionEnd(contents.Body, *in.Section)
	if err != nil {
		return 0, err
	}
	return at, file.AddCardUnder(*in.Section, card)
}

type cardEditInput struct {
	Path        string       `json:"path" jsonschema:"the deck the card is in"`
	Mark        string       `json:"mark" jsonschema:"the card's mark, as card_read gives it"`
	Values      []FieldValue `json:"values" jsonschema:"the fields to write, and what to put under each"`
	Stencil     string       `json:"stencil,omitempty" jsonschema:"the stencil it is cut by from now on, by the name card_stencil_list gave under name; left out, the card keeps the one it names"`
	Fingerprint string       `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func addCardEdit(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "card_edit",
		Title: "Edit a card",
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
	}, newHandler(func(ctx context.Context, in cardEditInput) (WriteOutcome, error) {
		return writeCardEdit(ctx, core, in)
	}))
}

func writeCardEdit(ctx context.Context, core Core, in cardEditInput) (WriteOutcome, error) {
	if err := checkCardSize(in.Values); err != nil {
		return WriteOutcome{}, err
	}
	written, _, err := changeDeck(ctx, core, in.Path, in.Fingerprint,
		func(contents cards.DeckContents, file *format.DeckFile) error {
			return setCardValues(contents, file, in)
		})
	return written, err
}

// setCardValues sets the stencil and values from the input on the card with the given mark.
func setCardValues(contents cards.DeckContents, file *format.DeckFile, in cardEditInput) error {
	card := domain.CardID(in.Mark)
	// The card must exist even when no value changes.
	if _, err := index(contents.Body.Cards, in.Mark); err != nil {
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
}

type cardValueRemoveInput struct {
	Path        string `json:"path" jsonschema:"the deck the card is in"`
	Mark        string `json:"mark" jsonschema:"the card's mark, as card_read gives it"`
	Field       string `json:"field" jsonschema:"the field to take off, as card_read gives it under field"`
	Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func addCardValueRemove(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "card_value_remove",
		Title: "Remove a card's field",
		Description: "Remove one field from one card: its heading, and what the person " +
			"wrote under it. `card_edit` writes a value and never takes one away, and an " +
			"empty value is a field standing empty, not a field gone. The fields around " +
			"it keep the order they were written in, and every other byte of the file is " +
			"left as it was. The stencil is not touched: it goes on declaring the field, " +
			"and every other card cut by it goes on carrying its own value. A card " +
			"writing nothing under that field is refused. The card is addressed by the " +
			"mark `card_read` gives it, the fingerprint from `card_read` is required, and " +
			"a write lands only on the deck that fingerprint names.",
	}, newHandler(func(ctx context.Context, in cardValueRemoveInput) (WriteOutcome, error) {
		written, _, err := changeDeck(ctx, core, in.Path, in.Fingerprint,
			func(_ cards.DeckContents, file *format.DeckFile) error {
				return file.RemoveValue(domain.CardID(in.Mark), in.Field)
			})
		return written, err
	}))
}

type cardRemoveInput struct {
	Path        string `json:"path" jsonschema:"the deck the card is in"`
	Mark        string `json:"mark" jsonschema:"the card's mark, as card_read gives it"`
	Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func addCardRemove(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "card_remove",
		Title: "Remove a card",
		Description: "Remove one card from a deck: its heading, the stencil it named and " +
			"every value under it. The rest of the file is left the bytes it was, down to " +
			"the blank lines and the whitespace at the ends of the lines; what every " +
			"write of a deck puts right is the mark of a card carrying none and the " +
			"heading of a card that has fallen out of step with its first field. " +
			"Nothing brings the card back, so read the deck before removing from it. The " +
			"fingerprint from `card_read` is required, and a write lands only on the deck " +
			"that fingerprint names.",
	}, newHandler(func(ctx context.Context, in cardRemoveInput) (WriteOutcome, error) {
		written, _, err := changeDeck(ctx, core, in.Path, in.Fingerprint,
			func(_ cards.DeckContents, file *format.DeckFile) error {
				return file.RemoveCard(domain.CardID(in.Mark))
			})
		return written, err
	}))
}

type cardSectionAddInput struct {
	Path        string `json:"path" jsonschema:"the deck to divide"`
	Name        string `json:"name" jsonschema:"what the section is called"`
	Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func addCardSectionAdd(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "card_section_add",
		Title: "Add a section",
		Description: "Write a section at the end of a deck. A section is a name a person " +
			"gives one run of a deck, and it is a name and nothing else: it carries no " +
			"field, no stencil and no schedule. `card_add` writes a card at the end of " +
			"one, and `card_read` says which section each card stands under. The " +
			"fingerprint from `card_read` is required, and a write lands only on the deck " +
			"that fingerprint names.",
	}, newHandler(func(ctx context.Context, in cardSectionAddInput) (WriteOutcome, error) {
		written, _, err := changeDeck(ctx, core, in.Path, in.Fingerprint,
			func(_ cards.DeckContents, file *format.DeckFile) error {
				return file.AddSection(format.Section{Name: in.Name})
			})
		return written, err
	}))
}

type cardSectionRenameInput struct {
	Path        string `json:"path" jsonschema:"the deck the section is in"`
	Section     int    `json:"section" jsonschema:"where the section stands in the deck's sections, counted from the first, as card_read gives them"`
	Name        string `json:"name" jsonschema:"what it is called from now on"`
	Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func addCardSectionRename(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "card_section_rename",
		Title: "Rename a section",
		Description: "Give one of a deck's sections a different name. Only the heading " +
			"line is written: the cards standing under it stay where they are, keep their " +
			"marks and keep what they hold, and what a person wrote beneath the heading is " +
			"left as it stands. The section is named by where it stands in the deck, " +
			"counted from the first, which is what `card_read` gives under `sections`. The " +
			"fingerprint from `card_read` is required, and a write lands only on the deck " +
			"that fingerprint names.",
	}, newHandler(func(ctx context.Context, in cardSectionRenameInput) (WriteOutcome, error) {
		written, _, err := changeDeck(ctx, core, in.Path, in.Fingerprint,
			func(_ cards.DeckContents, file *format.DeckFile) error {
				return file.RenameSection(in.Section, in.Name)
			})
		return written, err
	}))
}

type cardSectionRemoveInput struct {
	Path        string `json:"path" jsonschema:"the deck the section is in"`
	Section     int    `json:"section" jsonschema:"where the section stands in the deck's sections, counted from the first, as card_read gives them"`
	Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func addCardSectionRemove(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "card_section_remove",
		Title: "Remove a section",
		Description: "Remove the heading of one of a deck's sections. A section is a name " +
			"and nothing else, so this takes away the name alone: no card is removed, and " +
			"the cards that stood under the heading stay in the file, in the order they " +
			"were, under whatever section now stands above them. What a person wrote " +
			"beneath the heading stays there too. The section is named by where it stands " +
			"in the deck, counted from the first, which is what `card_read` gives under " +
			"`sections`. The fingerprint from `card_read` is required, and a write lands " +
			"only on the deck that fingerprint names.",
	}, newHandler(func(ctx context.Context, in cardSectionRemoveInput) (WriteOutcome, error) {
		written, _, err := changeDeck(ctx, core, in.Path, in.Fingerprint,
			func(_ cards.DeckContents, file *format.DeckFile) error {
				return file.RemoveSection(in.Section)
			})
		return written, err
	}))
}
