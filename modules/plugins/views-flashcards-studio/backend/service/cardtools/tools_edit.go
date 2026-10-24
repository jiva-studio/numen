package cardtools

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/tools"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/cards/deckedit"
	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/note"
)

// checkCardSize returns an error when the card values exceed the size limit of one write.
func checkCardSize(vs []FieldValue) error {
	if size := countBytes(vs); size > maxBytes {
		return fmt.Errorf("a card of %d bytes is more than this writes at once, which is %d", size, maxBytes)
	}
	return nil
}

// writeDeck presents the fingerprint the caller read to one edit of the deck at path.
func (t *Tools) writeDeck(
	ctx context.Context, path, fingerprint string, edit func(v domain.Vault, version deckedit.DeckVersion) (deckedit.EditResult, error),
) (WriteOutcome, error) {
	read, err := parseFingerprint(fingerprint)
	if err != nil {
		return WriteOutcome{}, err
	}
	v, err := t.getToolVault(ctx)
	if err != nil {
		return WriteOutcome{}, err
	}
	edited, err := edit(v, deckedit.DeckVersion{Path: path, Fingerprint: read})
	if err != nil {
		return WriteOutcome{}, err
	}
	return newWriteOutcome(path, edited), nil
}

type cardAddInput struct {
	Path        string       `json:"path" jsonschema:"the deck to write into"`
	Stencil     string       `json:"stencil" jsonschema:"the stencil it is cut by, by the name card_stencil_list gave under name"`
	Values      []FieldValue `json:"values" jsonschema:"what the card holds, in the order to write it"`
	Section     *int         `json:"section,omitempty" jsonschema:"which of the deck's sections to write it at the end of, counted from the first; left out, the card goes at the end of the deck"`
	Fingerprint string       `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func (t *Tools) addCardAdd(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Effect: tools.Writes, Servers: []tools.Server{tools.Full, tools.Review},
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
	}, t.writeCardAdd)
}

func (t *Tools) writeCardAdd(ctx context.Context, in cardAddInput) (WriteOutcome, error) {
	if err := checkCardSize(in.Values); err != nil {
		return WriteOutcome{}, err
	}
	var added deckedit.AddedCard
	content, err := t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
		var err error
		added, err = t.deckedit.AddCard.Execute(ctx, v, pos, in.Stencil, values(in.Values), in.Section)
		return added.EditResult, err
	})
	if err != nil {
		return WriteOutcome{}, err
	}
	content.Mark = added.Mark
	return content, nil
}

type cardEditInput struct {
	Path        string       `json:"path" jsonschema:"the deck the card is in"`
	Mark        string       `json:"mark" jsonschema:"the card's mark, as card_read gives it"`
	Values      []FieldValue `json:"values" jsonschema:"the fields to write, and what to put under each"`
	Stencil     string       `json:"stencil,omitempty" jsonschema:"the stencil it is cut by from now on, by the name card_stencil_list gave under name; left out, the card keeps the one it names"`
	Fingerprint string       `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func (t *Tools) addCardEdit(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Effect: tools.Writes, Servers: []tools.Server{tools.Full, tools.Review},
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
	}, t.writeCardEdit)
}

func (t *Tools) writeCardEdit(ctx context.Context, in cardEditInput) (WriteOutcome, error) {
	if err := checkCardSize(in.Values); err != nil {
		return WriteOutcome{}, err
	}
	return t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
		return t.deckedit.EditCard.Execute(ctx, v, pos, in.Mark, in.Stencil, values(in.Values))
	})
}

type cardValueRemoveInput struct {
	Path        string `json:"path" jsonschema:"the deck the card is in"`
	Mark        string `json:"mark" jsonschema:"the card's mark, as card_read gives it"`
	Field       string `json:"field" jsonschema:"the field to take off, as card_read gives it under field"`
	Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func (t *Tools) addCardValueRemove(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Effect: tools.Writes, Servers: []tools.Server{tools.Full, tools.Review}, Step: tools.Remove,
		Name:  "card_value_remove",
		Title: "Remove a card's field",
		Description: "Remove one field from one card: its heading, and what the person " +
			"wrote under it. `card_edit` writes a value and never takes one away, and an " +
			"empty value is a field standing empty, not a field gone. The fields around " +
			"it keep the order they were written in, and every other byte of the file is " +
			"left as it was. The stencil is not touched: it goes on declaring the field, " +
			"and every other card cut by it goes on carrying its own value. A card " +
			"writing nothing under that field fails. The card is addressed by the " +
			"mark `card_read` gives it, the fingerprint from `card_read` is required, and " +
			"a write lands only on the deck that fingerprint names.",
	}, func(ctx context.Context, in cardValueRemoveInput) (WriteOutcome, error) {
		return t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
			return t.deckedit.RemoveCardValue.Execute(ctx, v, pos, in.Mark, in.Field)
		})
	})
}

type cardRemoveInput struct {
	Path        string `json:"path" jsonschema:"the deck the card is in"`
	Mark        string `json:"mark" jsonschema:"the card's mark, as card_read gives it"`
	Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func (t *Tools) addCardRemove(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Effect: tools.Writes, Servers: []tools.Server{tools.Full, tools.Review}, Step: tools.Remove,
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
	}, func(ctx context.Context, in cardRemoveInput) (WriteOutcome, error) {
		return t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
			return t.deckedit.RemoveCard.Execute(ctx, v, pos, in.Mark)
		})
	})
}

type cardSectionAddInput struct {
	Path        string `json:"path" jsonschema:"the deck to divide"`
	Name        string `json:"name" jsonschema:"what the section is called"`
	Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func (t *Tools) addCardSectionAdd(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Effect: tools.Writes, Servers: []tools.Server{tools.Full, tools.Review},
		Name:  "card_section_add",
		Title: "Add a section",
		Description: "Write a section at the end of a deck. A section is a name a person " +
			"gives one run of a deck, and it is a name and nothing else: it carries no " +
			"field, no stencil and no schedule. `card_add` writes a card at the end of " +
			"one, and `card_read` says which section each card stands under. The " +
			"fingerprint from `card_read` is required, and a write lands only on the deck " +
			"that fingerprint names.",
	}, func(ctx context.Context, in cardSectionAddInput) (WriteOutcome, error) {
		return t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
			return t.deckedit.AddSection.Execute(ctx, v, pos, in.Name)
		})
	})
}

type cardSectionRenameInput struct {
	Path        string `json:"path" jsonschema:"the deck the section is in"`
	Section     int    `json:"section" jsonschema:"where the section stands in the deck's sections, counted from the first, as card_read gives them"`
	Name        string `json:"name" jsonschema:"what it is called from now on"`
	Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func (t *Tools) addCardSectionRename(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Effect: tools.Writes, Servers: []tools.Server{tools.Full, tools.Review}, Step: tools.Move,
		Name:  "card_section_rename",
		Title: "Rename a section",
		Description: "Give one of a deck's sections a different name. Only the heading " +
			"line is written: the cards standing under it stay where they are, keep their " +
			"marks and keep what they hold, and what a person wrote beneath the heading is " +
			"left as it stands. The section is named by where it stands in the deck, " +
			"counted from the first, which is what `card_read` gives under `sections`. The " +
			"fingerprint from `card_read` is required, and a write lands only on the deck " +
			"that fingerprint names.",
	}, func(ctx context.Context, in cardSectionRenameInput) (WriteOutcome, error) {
		return t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
			return t.deckedit.RenameSection.Execute(ctx, v, pos, in.Section, in.Name)
		})
	})
}

type cardSectionRemoveInput struct {
	Path        string `json:"path" jsonschema:"the deck the section is in"`
	Section     int    `json:"section" jsonschema:"where the section stands in the deck's sections, counted from the first, as card_read gives them"`
	Fingerprint string `json:"fingerprint" jsonschema:"what card_read said the deck was, which refuses a write over somebody else's edit"`
}

func (t *Tools) addCardSectionRemove(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Effect: tools.Writes, Servers: []tools.Server{tools.Full, tools.Review}, Step: tools.Remove,
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
	}, func(ctx context.Context, in cardSectionRemoveInput) (WriteOutcome, error) {
		return t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
			return t.deckedit.RemoveSection.Execute(ctx, v, pos, in.Section)
		})
	})
}

type stencilEditInput struct {
	Path        string   `json:"path" jsonschema:"the stencil to edit"`
	Fields      []string `json:"fields" jsonschema:"the names of the fields, in the order a person is asked for them; a card's heading is read from the first"`
	Faces       []Face   `json:"faces" jsonschema:"the ways a card cut by this stencil is shown"`
	Fingerprint string   `json:"fingerprint" jsonschema:"what card_stencil_read said the stencil was, which refuses a write over somebody else's edit"`
}

func (t *Tools) addCardStencilEdit(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Effect: tools.Writes, Servers: []tools.Server{tools.Full},
		Name:  "card_stencil_edit",
		Title: "Edit a stencil",
		Description: "Change a stencil in place: its fields and the faces a card cut by " +
			"it is shown through. Hand back the fingerprint from `card_stencil_read` so " +
			"the write fails if the stencil has changed in the meantime.",
	}, t.writeCardStencilEdit)
}

func (t *Tools) writeCardStencilEdit(ctx context.Context, in stencilEditInput) (WriteOutcome, error) {
	fp, err := parseFingerprint(in.Fingerprint)
	if err != nil {
		return WriteOutcome{}, err
	}
	v, err := t.getToolVault(ctx)
	if err != nil {
		return WriteOutcome{}, err
	}
	current, err := t.cards.Read.Stencil(ctx, v, in.Path)
	if err != nil {
		return WriteOutcome{}, err
	}
	if current.Outcome != note.Ok {
		if reason, ok := convertOutcomeToErrorCode(current.Outcome); ok {
			return WriteOutcome{}, errors.New(describeCode(reason))
		}
		return WriteOutcome{}, fmt.Errorf("%s: %s", in.Path, current.Outcome)
	}
	if current.Type != format.TypeStencil {
		return WriteOutcome{}, errors.New(describeCode(v1.ErrorCode_ERROR_CODE_NOT_A_STENCIL))
	}
	body, err := format.StencilBody(current.Body.Preamble, newFaceTemplates(in.Faces), current.Body.Tail)
	if err != nil {
		return WriteOutcome{}, err
	}
	newFp, err := t.cards.Write.Stencil(ctx, v, in.Path, body, in.Fields, fp)
	if err != nil {
		return WriteOutcome{}, err
	}
	return WriteOutcome{
		Path:        in.Path,
		Fingerprint: fingerprintOf(newFp),
	}, nil
}
