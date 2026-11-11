package cardtools

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards/deckedit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
)

func checkCardSize(vs []FieldValue) error {
	if size := countBytes(vs); size > maxBytes {
		return fmt.Errorf("a card of %d bytes is more than this writes at once, which is %d", size, maxBytes)
	}
	return nil
}

func (t *Tools) writeDeck(
	ctx context.Context, path, stat string, edit func(v domain.Vault, version deckedit.DeckVersion) (deckedit.EditResult, error),
) (WriteOutcome, error) {
	read, err := parseFingerprint(stat)
	if err != nil {
		return WriteOutcome{}, err
	}
	v, err := t.getToolVault(ctx)
	if err != nil {
		return WriteOutcome{}, err
	}
	edited, err := edit(v, deckedit.DeckVersion{Path: path, FileStat: read})
	if err != nil {
		return WriteOutcome{}, err
	}
	return newWriteOutcome(path, edited), nil
}

type cardAddInput struct {
	Path         string       `json:"path" jsonschema:"the path of the deck to write into, relative to the vault folder"`
	CardModel    string       `json:"cardModel" jsonschema:"the card model of the card, by the name card_model_list returns under name"`
	Values       []FieldValue `json:"values" jsonschema:"the field values, in the order to write them"`
	SectionIndex *int         `json:"sectionIndex,omitempty" jsonschema:"the index of the section to append the card to, counted from 0; when left out, the card is appended to the deck"`
	Fingerprint  string       `json:"fingerprint" jsonschema:"the deck fingerprint returned by card_read; a write against a changed deck is rejected"`
}

func (t *Tools) addCardAdd(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:   tools.ByEffect,
		Effect: tools.ReadWrite, Servers: []tools.Server{tools.Full},
		Name:  "card_add",
		Title: "Add a card",
		Description: "Append one card to a deck, or to one of its sections. Name its card model as " +
			"`card_model_list` returns it under `name`, and give a value for every field that " +
			"card model declares, the first included. Do not write cards into the deck file " +
			"directly: a card written by hand without the wikilink under its heading has no card " +
			"model, and no error reports it. The heading of the card is written from its first " +
			"field, and the new card id is returned under `cardId`. " +
			"The fingerprint from `card_read` is required; a write against a changed deck is rejected.",
	}, t.writeCardAdd)
}

func (t *Tools) writeCardAdd(ctx context.Context, in cardAddInput) (WriteOutcome, error) {
	if err := checkCardSize(in.Values); err != nil {
		return WriteOutcome{}, err
	}
	var added deckedit.AddedCard
	content, err := t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
		var err error
		added, err = t.deckCards.AddCard.Execute(ctx, v, pos, in.CardModel, values(in.Values), in.SectionIndex)
		return added.EditResult, err
	})
	if err != nil {
		return WriteOutcome{}, err
	}
	content.CardID = added.CardID
	return content, nil
}

type cardEditInput struct {
	Path        string       `json:"path" jsonschema:"the path of the deck of the card, relative to the vault folder"`
	CardID      string       `json:"cardId" jsonschema:"the card id, as card_read returns it"`
	Values      []FieldValue `json:"values" jsonschema:"the field values to write"`
	CardModel   string       `json:"cardModel,omitempty" jsonschema:"the new card model of the card, by the name card_model_list returns under name; when left out, the card model is unchanged"`
	Fingerprint string       `json:"fingerprint" jsonschema:"the deck fingerprint returned by card_read; a write against a changed deck is rejected"`
}

func (t *Tools) addCardEdit(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:   tools.ByEffect,
		Effect: tools.ReadWrite, Servers: []tools.Server{tools.Full},
		Name:  "card_edit",
		Title: "Edit a card",
		Description: "Write field values into one card of a deck. A field the card has is replaced, " +
			"a field it lacks is appended to the card, and fields left out of the call are " +
			"unchanged. Send only the fields to change; `card_value_remove` removes a field. " +
			"The card is addressed by the card id `card_read` returns, which stays the same for " +
			"the life of the card, so data attached to the card stays attached after any edit. " +
			"The first field is written like any other, and the heading of the card is updated " +
			"from it. " +
			"The fingerprint from `card_read` is required; a write against a changed deck is rejected.",
	}, t.writeCardEdit)
}

func (t *Tools) writeCardEdit(ctx context.Context, in cardEditInput) (WriteOutcome, error) {
	if err := checkCardSize(in.Values); err != nil {
		return WriteOutcome{}, err
	}
	return t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
		return t.deckCards.EditCard.Execute(ctx, v, pos, in.CardID, in.CardModel, values(in.Values))
	})
}

type cardValueRemoveInput struct {
	Path        string `json:"path" jsonschema:"the path of the deck of the card, relative to the vault folder"`
	CardID      string `json:"cardId" jsonschema:"the card id, as card_read returns it"`
	Field       string `json:"field" jsonschema:"the field to remove, as card_read returns it under field"`
	Fingerprint string `json:"fingerprint" jsonschema:"the deck fingerprint returned by card_read; a write against a changed deck is rejected"`
}

func (t *Tools) addCardValueRemove(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Effect: tools.ReadWrite, Servers: []tools.Server{tools.Full}, Step: tools.Remove,
		Name:  "card_value_remove",
		Title: "Remove a card's field",
		Description: "Remove one field from one card: its heading and its value. `card_edit` writes " +
			"values and never removes one, and an empty value is an empty field, not a removed " +
			"one. The other fields keep their order, and the rest of the file is unchanged. " +
			"The card model is unchanged: it still declares the field, and the other cards of " +
			"the card model keep their values. The call fails for a card without that field. " +
			"The card is addressed by the card id `card_read` returns. " +
			"The fingerprint from `card_read` is required; a write against a changed deck is rejected.",
	}, func(ctx context.Context, in cardValueRemoveInput) (WriteOutcome, error) {
		return t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
			return t.deckCards.RemoveCardValue.Execute(ctx, v, pos, in.CardID, in.Field)
		})
	})
}

type cardRemoveInput struct {
	Path        string `json:"path" jsonschema:"the path of the deck of the card, relative to the vault folder"`
	CardID      string `json:"cardId" jsonschema:"the card id, as card_read returns it"`
	Fingerprint string `json:"fingerprint" jsonschema:"the deck fingerprint returned by card_read; a write against a changed deck is rejected"`
}

func (t *Tools) addCardRemove(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Effect: tools.ReadWrite, Servers: []tools.Server{tools.Full}, Step: tools.Remove,
		Name:  "card_remove",
		Title: "Remove a card",
		Description: "Remove one card from a deck: its heading, its card model link and every value. " +
			"The rest of the file is unchanged byte for byte, blank lines and trailing " +
			"whitespace included; every deck write adds a missing card id and rewrites a " +
			"heading that no longer matches its first field. A removed card cannot be restored, " +
			"so read the deck before removing from it. " +
			"The fingerprint from `card_read` is required; a write against a changed deck is rejected.",
	}, func(ctx context.Context, in cardRemoveInput) (WriteOutcome, error) {
		return t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
			return t.deckCards.RemoveCard.Execute(ctx, v, pos, in.CardID)
		})
	})
}

type cardSectionAddInput struct {
	Path        string `json:"path" jsonschema:"the path of the deck, relative to the vault folder"`
	Name        string `json:"name" jsonschema:"the section name"`
	Fingerprint string `json:"fingerprint" jsonschema:"the deck fingerprint returned by card_read; a write against a changed deck is rejected"`
}

func (t *Tools) addCardSectionAdd(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:   tools.ByEffect,
		Effect: tools.ReadWrite, Servers: []tools.Server{tools.Full},
		Name:  "card_section_add",
		Title: "Add a section",
		Description: "Append a section to a deck. A section is a name for a range of cards and has " +
			"no other content: no field, no card model and no schedule. `card_add` appends a card to " +
			"a section, and `card_read` returns the section index of each card. " +
			"The fingerprint from `card_read` is required; a write against a changed deck is rejected.",
	}, func(ctx context.Context, in cardSectionAddInput) (WriteOutcome, error) {
		return t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
			return t.deckSections.AddSection.Execute(ctx, v, pos, in.Name)
		})
	})
}

type cardSectionRenameInput struct {
	Path         string `json:"path" jsonschema:"the path of the deck of the section, relative to the vault folder"`
	SectionIndex int    `json:"sectionIndex" jsonschema:"the index of the section in the deck, counted from 0, as card_read returns the sections"`
	Name         string `json:"name" jsonschema:"the new section name"`
	Fingerprint  string `json:"fingerprint" jsonschema:"the deck fingerprint returned by card_read; a write against a changed deck is rejected"`
}

func (t *Tools) addCardSectionRename(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Effect: tools.ReadWrite, Servers: []tools.Server{tools.Full}, Step: tools.Move,
		Name:  "card_section_rename",
		Title: "Rename a section",
		Description: "Rename one section of a deck. Only the heading line is written: the cards under " +
			"it keep their position, card ids and values, and text under the heading is " +
			"unchanged. The section is addressed by its index in the deck, counted from 0, as " +
			"`card_read` returns `sections`. " +
			"The fingerprint from `card_read` is required; a write against a changed deck is rejected.",
	}, func(ctx context.Context, in cardSectionRenameInput) (WriteOutcome, error) {
		return t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
			return t.deckSections.RenameSection.Execute(ctx, v, pos, in.SectionIndex, in.Name)
		})
	})
}

type cardSectionRemoveInput struct {
	Path         string `json:"path" jsonschema:"the path of the deck of the section, relative to the vault folder"`
	SectionIndex int    `json:"sectionIndex" jsonschema:"the index of the section in the deck, counted from 0, as card_read returns the sections"`
	Fingerprint  string `json:"fingerprint" jsonschema:"the deck fingerprint returned by card_read; a write against a changed deck is rejected"`
}

func (t *Tools) addCardSectionRemove(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Effect: tools.ReadWrite, Servers: []tools.Server{tools.Full}, Step: tools.Remove,
		Name:  "card_section_remove",
		Title: "Remove a section",
		Description: "Remove the heading of one section of a deck. A section is only a name, so only " +
			"the name is removed: no card is removed, and the cards under the heading stay in the " +
			"file in their order, under the preceding section. Text under the heading stays too. " +
			"The section is addressed by its index in the deck, counted from 0, as `card_read` " +
			"returns `sections`. " +
			"The fingerprint from `card_read` is required; a write against a changed deck is rejected.",
	}, func(ctx context.Context, in cardSectionRemoveInput) (WriteOutcome, error) {
		return t.writeDeck(ctx, in.Path, in.Fingerprint, func(v domain.Vault, pos deckedit.DeckVersion) (deckedit.EditResult, error) {
			return t.deckSections.RemoveSection.Execute(ctx, v, pos, in.SectionIndex)
		})
	})
}

type cardModelEditInput struct {
	Path        string         `json:"path" jsonschema:"the path of the card model to edit, relative to the vault folder"`
	Fields      []string       `json:"fields" jsonschema:"the field names, in order; the heading of a card is taken from the first field"`
	Templates   []CardTemplate `json:"templates" jsonschema:"the templates that render the cards of this card model"`
	Fingerprint string         `json:"fingerprint" jsonschema:"the card model fingerprint returned by card_model_read; a write against a changed card model is rejected"`
}

func (t *Tools) addCardCardModelEdit(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:   tools.ByEffect,
		Effect: tools.ReadWrite, Servers: []tools.Server{tools.Full},
		Name:  "card_model_edit",
		Title: "Edit a card model",
		Description: "Change the field names and templates of a card model in place. Pass the " +
			"fingerprint from `card_model_read`; a write against a changed card model is rejected.",
	}, t.writeCardCardModelEdit)
}

func (t *Tools) writeCardCardModelEdit(ctx context.Context, in cardModelEditInput) (WriteOutcome, error) {
	fp, err := parseFingerprint(in.Fingerprint)
	if err != nil {
		return WriteOutcome{}, err
	}
	v, err := t.getToolVault(ctx)
	if err != nil {
		return WriteOutcome{}, err
	}
	current, err := t.cards.Read.CardModel(ctx, v, in.Path)
	if err != nil {
		return WriteOutcome{}, err
	}
	if current.Outcome != note.Ok {
		// gomutants:disable-next-line BRANCH_IF reason="All non-Ok ReadOutcome values map to an ErrorCode in convertOutcomeToErrorCode."
		if reason, ok := convertOutcomeToErrorCode(current.Outcome); ok {
			return WriteOutcome{}, errors.New(describeCode(reason))
		}
		return WriteOutcome{}, fmt.Errorf("%s: %s", in.Path, current.Outcome)
	}
	if current.Type != format.TypeCardModel {
		return WriteOutcome{}, errors.New(describeCode(api.ErrorCodeNotACardModel))
	}
	body, err := format.CardModelBody(current.Body.Preamble, newCardTemplates(in.Templates), current.Body.Tail)
	// gomutants:disable-next-line BRANCH_IF reason="CardModelBody on in-memory strings does not fail."
	if err != nil {
		return WriteOutcome{}, err
	}
	newFp, err := t.cards.Write.CardModel(ctx, v, in.Path, body, in.Fields, fp)
	if err != nil {
		return WriteOutcome{}, err
	}
	return WriteOutcome{
		Path:        in.Path,
		Fingerprint: fingerprintOf(newFp),
		Total:       0,
		CardID:      "",
	}, nil
}
