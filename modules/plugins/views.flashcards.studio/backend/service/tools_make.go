package service

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cards"
)

type deckCreateInput struct {
	Title  string `json:"title" jsonschema:"what the deck is called"`
	Folder string `json:"folder,omitempty" jsonschema:"where to file it, relative to the vault folder; the root by default"`
}

func (b *Backend) addCardDeckCreate(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:  "card_deck_create",
		Title: "Create a deck",
		Description: "Make a deck of no cards, and fill it with `card_add`. A deck is a " +
			"file of cards and says so from the moment it exists, so it is a deck to " +
			"everything that reads the vault before a card is written into it. Make a " +
			"deck here: a note made with `note_create` is an ordinary note, and " +
			"`card_add` writes into a deck alone. The deck is named after its title.",
		Effect: tools.Writes, Servers: []tools.Server{tools.Full},
	}, func(ctx context.Context, in deckCreateInput) (cards.CreateNoteResult, error) {
		v, err := b.getToolVault(ctx)
		if err != nil {
			return cards.CreateNoteResult{}, err
		}
		return b.scenarios().cards.Create.Deck(ctx, v, cards.New{Title: in.Title, Path: in.Folder})
	})
}

type stencilCreateInput struct {
	Title  string   `json:"title" jsonschema:"what the stencil is called"`
	Fields []string `json:"fields" jsonschema:"the names of the fields, in the order a person is asked for them; a card's heading is read from the first"`
	Faces  []Face   `json:"faces" jsonschema:"the ways a card cut by this stencil is shown"`
	Folder string   `json:"folder,omitempty" jsonschema:"where to file it, relative to the vault folder; the root by default"`
}

func (b *Backend) addCardStencilCreate(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:  "card_stencil_create",
		Title: "Create a stencil",
		Description: "Make a stencil: the fields a card is asked for, in the order to ask " +
			"for them, and the faces one is shown through. A face is HTML with " +
			"`{{Field}}` standing where a value goes, and every name in braces must be " +
			"one of the fields. A card's heading is read from the first field, so there " +
			"is at least one; it holds as many lines as a person writes, and the " +
			"heading is its first line. The stencil is filed under its title, and the " +
			"name a card's wikilink writes is that filename, which `card_stencil_list` " +
			"gives under `name`.",
		Effect: tools.Writes, Servers: []tools.Server{tools.Full},
	}, func(ctx context.Context, in stencilCreateInput) (cards.CreateNoteResult, error) {
		v, err := b.getToolVault(ctx)
		if err != nil {
			return cards.CreateNoteResult{}, err
		}
		return b.scenarios().cards.Create.Stencil(ctx, v, cards.New{
			Title: in.Title, Path: in.Folder, Fields: in.Fields, Faces: newFaceTemplates(in.Faces),
		})
	})
}

type fieldRenameInput struct {
	Path string `json:"path" jsonschema:"the stencil that declares the field"`
	From string `json:"from" jsonschema:"the field's name now"`
	To   string `json:"to" jsonschema:"what it is called from now on"`
}

type fieldRenameOutput struct {
	Decks      []string `json:"decks,omitempty" jsonschema:"the decks a heading was rewritten in"`
	Cards      int      `json:"cards" jsonschema:"how many headings were rewritten"`
	NotWritten []string `json:"notWritten,omitempty" jsonschema:"the decks the rename could not be written to, which keep the old heading"`
}

func (b *Backend) addCardFieldRename(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:  "card_field_rename",
		Title: "Rename a field",
		Description: "Give one of a stencil's fields a different name, everywhere it is " +
			"written. A field's name stands in the stencil that declares it and as a " +
			"heading in every card of every deck that stencil cuts, so renaming it in " +
			"one place alone leaves values under a heading nothing declares. This " +
			"rewrites them all and leaves what stands under each heading as it was. " +
			"Every field a stencil declares stands under its own heading in every card, " +
			"the first included, so renaming any of them reaches the decks. A " +
			"deck it could not be written to comes back under `notWritten` and keeps the " +
			"old heading. Every deck of the vault is read for this, so a vault holding " +
			"more than " + fmt.Sprint(cards.MaxDecksPerRename) + " of them is refused whole and the " +
			"person renames the field in the window.",
		Effect: tools.Writes, Servers: []tools.Server{tools.Full}, Step: tools.Move,
	}, b.renameCardField)
}

func (b *Backend) renameCardField(ctx context.Context, in fieldRenameInput) (fieldRenameOutput, error) {
	v, err := b.getToolVault(ctx)
	if err != nil {
		return fieldRenameOutput{}, err
	}
	renamed, err := b.scenarios().cards.BoundedRenameField.Execute(ctx, v, cards.Rename{Stencil: in.Path, From: in.From, To: in.To})
	if err != nil {
		return fieldRenameOutput{}, err
	}
	res := fieldRenameOutput{Decks: renamed.Decks, Cards: renamed.Cards}
	for _, deck := range renamed.NotWritten {
		res.NotWritten = append(res.NotWritten, deck.Path)
	}
	return res, nil
}
