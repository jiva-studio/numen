package cardtools

import (
	"context"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
)

type deckCreateInput struct {
	Title  string `json:"title" jsonschema:"the title of the deck"`
	Folder string `json:"folder,omitempty" jsonschema:"the folder to create it in, relative to the vault folder; the root by default"`
}

func (t *Tools) addCardDeckCreate(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "card_deck_create",
		Title: "Create a deck",
		Description: "Create an empty deck; add cards with `card_add`. A deck is a file of " +
			"cards and is marked as a deck from creation, before it has any card. Create decks " +
			"here: a note created with `note_create` is an ordinary note, and `card_add` writes " +
			"only into a deck. The file is named after the title.",
		Effect: tools.ReadWrite, Servers: []tools.Server{tools.Full},
	}, func(ctx context.Context, in deckCreateInput) (NoteCreated, error) {
		v, err := t.getToolVault(ctx)
		if err != nil {
			return NoteCreated{}, err
		}
		return newNoteCreated(t.cards.Create.Deck(ctx, v, cards.New{Title: in.Title, Body: "", Path: in.Folder, Fields: nil, Templates: nil}))
	})
}

type cardModelCreateInput struct {
	Title     string         `json:"title" jsonschema:"the title of the card model"`
	Fields    []string       `json:"fields" jsonschema:"the field names, in order; the heading of a card is taken from the first field"`
	Templates []CardTemplate `json:"templates" jsonschema:"the templates that render the cards of this card model"`
	Folder    string         `json:"folder,omitempty" jsonschema:"the folder to create it in, relative to the vault folder; the root by default"`
}

func (t *Tools) addCardCardModelCreate(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "card_model_create",
		Title: "Create a card model",
		Description: "Create a card model: its field names in order, and the templates its " +
			"cards are rendered with. A template is HTML with `{{Field}}` placeholders, and " +
			"every name in braces must be one of the fields. The heading of a card is the first " +
			"line of its first field, so at least one field is required. The file is named after " +
			"the title, and that filename is the name used in a card's wikilink, which " +
			"`card_model_list` returns under `name`.",
		Effect: tools.ReadWrite, Servers: []tools.Server{tools.Full},
	}, func(ctx context.Context, in cardModelCreateInput) (NoteCreated, error) {
		v, err := t.getToolVault(ctx)
		if err != nil {
			return NoteCreated{}, err
		}
		return newNoteCreated(t.cards.Create.CardModel(ctx, v, cards.New{
			Title: in.Title, Body: "", Path: in.Folder, Fields: in.Fields, Templates: newCardTemplates(in.Templates),
		}))
	})
}

type fieldRenameInput struct {
	Path string `json:"path" jsonschema:"the path of the card model that declares the field"`
	From string `json:"from" jsonschema:"the current field name"`
	To   string `json:"to" jsonschema:"the new field name"`
}

type fieldRenameOutput struct {
	UpdatedDecks    []string `json:"updatedDecks,omitempty" jsonschema:"the decks in which a heading was rewritten"`
	UpdatedHeadings int      `json:"updatedHeadings" jsonschema:"the number of rewritten headings"`
	FailedDecks     []string `json:"failedDecks,omitempty" jsonschema:"the decks the rename could not be written to; they keep the old heading"`
}

func (t *Tools) addCardFieldRename(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:  "card_field_rename",
		Title: "Rename a field",
		Description: "Rename a field of a card model everywhere it is written. A field name is " +
			"in the card model that declares it and is a heading in every card of that card " +
			"model in every deck, so renaming it in one place leaves values under a heading no " +
			"card model declares. This rewrites every heading and keeps the values under them. " +
			"Every field, the first included, is a heading in every card, so any rename updates " +
			"the decks. A deck that could not be written is returned under `failedDecks` and " +
			"keeps the old heading. Every deck of the vault is read, so the call fails for a " +
			"vault with more than " + fmt.Sprint(cards.MaxDecksPerRename) + " decks; rename the " +
			"field in the window then.",
		Effect: tools.ReadWrite, Servers: []tools.Server{tools.Full}, Step: tools.Move,
	}, t.renameCardField)
}

func (t *Tools) renameCardField(ctx context.Context, in fieldRenameInput) (fieldRenameOutput, error) {
	v, err := t.getToolVault(ctx)
	if err != nil {
		return fieldRenameOutput{}, err
	}
	renamed, err := t.bounded.Execute(ctx, v, cards.Rename{
		CardModel: in.Path, From: in.From, To: in.To, FileStat: domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}},
	})
	if err != nil {
		return fieldRenameOutput{}, err
	}
	res := fieldRenameOutput{UpdatedDecks: renamed.Decks, UpdatedHeadings: renamed.Cards, FailedDecks: nil}
	for _, deck := range renamed.NotWritten {
		res.FailedDecks = append(res.FailedDecks, deck.Path)
	}
	return res, nil
}
