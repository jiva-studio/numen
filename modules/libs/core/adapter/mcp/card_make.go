package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
)

func addCardWriteTools(server *sdk.Server, core Core) {
	addCardEditTools(server, core)
	addCardCreateTools(server, core)
}

// addCardCreateTools adds the tools that create decks and stencils and rename stencil fields.
func addCardCreateTools(server *sdk.Server, core Core) {
	addCardDeckCreate(server, core)
	addCardStencilCreate(server, core)
	addCardFieldRename(server, core)
}

type deckCreateInput struct {
	Title  string `json:"title" jsonschema:"what the deck is called"`
	Folder string `json:"folder,omitempty" jsonschema:"where to file it, relative to the vault folder; the root by default"`
}

func addCardDeckCreate(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "card_deck_create",
		Title: "Create a deck",
		Description: "Make a deck of no cards, and fill it with `card_add`. A deck is a " +
			"file of cards and says so from the moment it exists, so it is a deck to " +
			"everything that reads the vault before a card is written into it. Make a " +
			"deck here: a note made with `note_create` is an ordinary note, and " +
			"`card_add` writes into a deck alone. The deck is named after its title.",
	}, newHandler(func(ctx context.Context, in deckCreateInput) (cards.CreateNoteResult, error) {
		return core.Cards.Create.Deck(ctx, core.getCurrentVault().Vault, cards.New{
			Title: in.Title, Path: in.Folder,
		})
	}))
}

type stencilCreateInput struct {
	Title  string   `json:"title" jsonschema:"what the stencil is called"`
	Fields []string `json:"fields" jsonschema:"the names of the fields, in the order a person is asked for them; a card's heading is read from the first"`
	Faces  []Face   `json:"faces" jsonschema:"the ways a card cut by this stencil is shown"`
	Folder string   `json:"folder,omitempty" jsonschema:"where to file it, relative to the vault folder; the root by default"`
}

func addCardStencilCreate(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
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
	}, newHandler(func(ctx context.Context, in stencilCreateInput) (cards.CreateNoteResult, error) {
		return createCardStencil(ctx, core, in)
	}))
}

func createCardStencil(ctx context.Context, core Core, in stencilCreateInput) (cards.CreateNoteResult, error) {
	return core.Cards.Create.Stencil(ctx, core.getCurrentVault().Vault, cards.New{
		Title: in.Title, Path: in.Folder, Fields: in.Fields, Faces: newFaceTemplates(in.Faces),
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

func addCardFieldRename(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
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
	}, newHandler(func(ctx context.Context, in fieldRenameInput) (fieldRenameOutput, error) {
		return renameCardField(ctx, core, in)
	}))
}

func renameCardField(ctx context.Context, core Core, in fieldRenameInput) (fieldRenameOutput, error) {
	v := core.getCurrentVault().Vault
	renamed, err := core.Cards.RenameField.Execute(ctx, v, cards.Rename{
		Stencil: in.Path, From: in.From, To: in.To,
	})
	if err != nil {
		return fieldRenameOutput{}, err
	}
	res := fieldRenameOutput{Decks: renamed.Decks, Cards: renamed.Cards}
	for _, deck := range renamed.NotWritten {
		res.NotWritten = append(res.NotWritten, deck.Path)
	}
	return res, nil
}

// Face is one way a stencil shows a card.
type Face struct {
	Name  string `json:"name" jsonschema:"what the face is called"`
	Front string `json:"front" jsonschema:"what is shown before the answer, as HTML with {{Field}} where a value goes"`
	Back  string `json:"back" jsonschema:"what is shown after it"`
}

func newFaceTemplates(fs []Face) []format.FaceTemplate {
	out := make([]format.FaceTemplate, 0, len(fs))
	for _, f := range fs {
		out = append(out, format.FaceTemplate{Name: f.Name, Front: f.Front, Back: f.Back})
	}
	return out
}

func newFaces(templates []format.FaceTemplate) []Face {
	out := make([]Face, 0, len(templates))
	for _, t := range templates {
		out = append(out, Face{Name: t.Name, Front: t.Front, Back: t.Back})
	}
	return out
}
