package cardtools

import (
	"context"
	"errors"
	"fmt"

	flashcardsv1 "github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards/deckedit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
)

// addCardCurrent adds the tool that returns the card the person is looking at.
func (t *Tools) addCardCurrent(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:  "card_current",
		Title: "Show the card in front",
		Description: "The card in front of the person: the deck it stands in, the card id " +
			"`card_read` addresses it by, and the template it is being shown through. Ask it " +
			"before saying anything about the card they are on — a person answers one " +
			"card and moves to the next while you work — and read the card itself with " +
			"`card_read`. Nothing is in front of them between cards, and the deck comes " +
			"back empty.",
		Effect: tools.ReadOnly, Servers: []tools.Server{tools.Full, tools.Reading},
	}, func(ctx context.Context, _ struct{}) (CurrentCard, error) {
		request := tools.GetCallContext(ctx)
		return CurrentCard{Deck: request[askedDeck], Card: request[askedCard], Template: request[askedTemplate]}, nil
	})
}

type cardModelListInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"how many card models to return, at most 50, which is also the default"`
}

type cardModelListOutput struct {
	CardModels []CardModel `json:"cardModels"`
	Total      int         `json:"total" jsonschema:"how many card models the vault holds, which stands above the list when the ceiling was reached"`
}

func (t *Tools) addCardCardModelList(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:  "card_model_list",
		Title: "List card models",
		Description: "The card models of the vault: where each is filed, the name a card's " +
			"wikilink writes, what it is called, and what a card cut by it is asked for. " +
			"Call this before writing a card, because a card names its card model and asks " +
			"for the fields that card model declares. At most 50 come back, and `total` says " +
			"how many the vault holds, so a list shorter than that was cut by the " +
			"ceiling. A card model with no fields listed is one that could not be read.",
		Effect: tools.ReadOnly, Servers: []tools.Server{tools.Full, tools.Reading},
	}, t.listCardCardModels)
}

func (t *Tools) listCardCardModels(ctx context.Context, in cardModelListInput) (cardModelListOutput, error) {
	if in.Limit > maxCardModels {
		return cardModelListOutput{}, fmt.Errorf("ask for at most %d card models at a time", maxCardModels)
	}
	limit := maxCardModels
	if in.Limit > 0 {
		limit = in.Limit
	}
	v, err := t.getToolVault(ctx)
	if err != nil {
		return cardModelListOutput{}, err
	}
	cardModels, count, err := t.cards.List.Execute(ctx, v, limit)
	if err != nil {
		return cardModelListOutput{}, err
	}
	res := cardModelListOutput{Total: count}
	for _, s := range cardModels {
		res.CardModels = append(res.CardModels, CardModel{
			Path: s.Path, Name: domain.Basename(s.Path), Title: s.Title, Fields: s.Fields,
		})
	}
	return res, nil
}

type cardModelReadInput struct {
	Path string `json:"path" jsonschema:"the card model to read"`
}

type cardModelReadOutput struct {
	Path        string         `json:"path"`
	Title       string         `json:"title"`
	Fields      []string       `json:"fields"`
	Templates   []CardTemplate `json:"templates"`
	Fingerprint string         `json:"fingerprint" jsonschema:"hand this to card_model_edit to refuse a write over an edit you did not see"`
}

func (t *Tools) addCardCardModelRead(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:  "card_model_read",
		Title: "Read a card model",
		Description: "The fields and templates of one card model: what it is called, the " +
			"fields a card cut by it is asked for, and the ways a card is shown through " +
			"it. Hand the fingerprint back to `card_model_edit` to edit the card model in place.",
		Effect: tools.ReadOnly, Servers: []tools.Server{tools.Full, tools.Reading},
	}, t.readCardCardModel)
}

func (t *Tools) readCardCardModel(ctx context.Context, in cardModelReadInput) (cardModelReadOutput, error) {
	v, err := t.getToolVault(ctx)
	if err != nil {
		return cardModelReadOutput{}, err
	}
	s, err := t.cards.Read.CardModel(ctx, v, in.Path)
	if err != nil {
		return cardModelReadOutput{}, err
	}
	if s.Outcome != note.Ok {
		if reason, ok := convertOutcomeToErrorCode(s.Outcome); ok {
			return cardModelReadOutput{}, errors.New(describeCode(reason))
		}
		return cardModelReadOutput{}, fmt.Errorf("%s: %s", in.Path, s.Outcome)
	}
	if s.Type != format.TypeCardModel {
		return cardModelReadOutput{}, errors.New(describeCode(flashcardsv1.ErrorCode_ERROR_CODE_NOT_A_CARD_MODEL))
	}
	title := s.Title
	if title == "" {
		title = domain.Basename(in.Path)
	}
	fields := s.Body.Fields
	if fields == nil {
		fields = []string{}
	}
	return cardModelReadOutput{
		Path:        in.Path,
		Title:       title,
		Fields:      fields,
		Templates:   newTemplates(s.Body.Templates),
		Fingerprint: fingerprintOf(s.FileStat),
	}, nil
}

type cardReadInput struct {
	Path   string   `json:"path" jsonschema:"the deck to read"`
	From   int      `json:"from,omitempty" jsonschema:"which card to start at, counted from zero"`
	Limit  int      `json:"limit,omitempty" jsonschema:"how many cards to return"`
	CardID string   `json:"cardId,omitempty" jsonschema:"one card's id, to read that card alone; from and limit say nothing about a call naming it"`
	Fields []string `json:"fields,omitempty" jsonschema:"the fields to answer with, spelled as the card writes them, which is what card_read gives; every other value is left out, and a card writing none of them comes back holding nothing"`
}

type cardReadOutput struct {
	Cards       []Card   `json:"cards"`
	Sections    []string `json:"sections,omitempty" jsonschema:"what the deck's sections are called, in the order they stand in the file"`
	Total       int      `json:"total" jsonschema:"how many cards the deck holds"`
	Faults      []Fault  `json:"faults,omitempty"`
	Fingerprint string   `json:"fingerprint" jsonschema:"hand this to a writing tool to refuse a write over an edit you did not see"`
}

func (t *Tools) addCardRead(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:  "card_read",
		Title: "Read cards",
		Description: "The cards of one deck, in the order they stand in the file, each " +
			"under the card id every other tool here addresses it by. A deck holds as many " +
			"cards as a person writes, so this answers a run of them at a time: `total` " +
			"says how many there are, and `from` takes the next run. The sections are " +
			"the runs a person divided the deck into, and every card says which of them " +
			"it stands under. What was wrong with the file comes back under `faults`, " +
			"on the card it is against. The fingerprint is what the writing tools want: " +
			"hand it back and a write fails if the person changed the deck in the " +
			"meantime.",
		Effect: tools.ReadOnly, Servers: []tools.Server{tools.Full, tools.Reading},
	}, t.readCards)
}

func (t *Tools) readCards(ctx context.Context, in cardReadInput) (cardReadOutput, error) {
	// The limit applies to the number of cards one call reads.
	if in.CardID == "" && in.Limit > maxCards {
		return cardReadOutput{}, fmt.Errorf("read at most %d cards at a time", maxCards)
	}
	limit := in.Limit
	if limit <= 0 {
		limit = maxCards
	}
	v, err := t.getToolVault(ctx)
	if err != nil {
		return cardReadOutput{}, err
	}
	read, err := t.deckedit.GetCardPage.Execute(ctx, v, deckedit.Query{
		Path: in.Path, From: in.From, Limit: limit, CardID: in.CardID, Fields: in.Fields,
	})
	if err != nil {
		return cardReadOutput{}, err
	}
	res := cardReadOutput{
		Sections:    read.Sections,
		Total:       read.Total,
		Faults:      faults(read.Problems),
		Fingerprint: fingerprintOf(read.FileStat),
	}
	for _, card := range read.Cards {
		res.Cards = append(res.Cards, newCard(card))
	}
	return res, nil
}
