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

func (t *Tools) addCardCurrentGet(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "card_current_get",
		Title: "Get the open card",
		Description: "Get the card open in the review window: the path of its deck, the card id " +
			"`card_read` takes, and the name of the template it is shown with. Call it " +
			"before describing the open card, because the user moves to the next card while " +
			"you work, and read the card itself with `card_read`. Between cards no card is " +
			"open, and `path` is empty.",
		Effect: tools.ReadOnly, Servers: []tools.Server{tools.Full, tools.Reading},
	}, func(ctx context.Context, _ struct{}) (CurrentCard, error) {
		request := tools.GetCallContext(ctx)
		return CurrentCard{Path: request[requestedDeck], CardID: request[requestedCard], TemplateName: request[requestedTemplate]}, nil
	})
}

type cardModelListInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"how many card models to return, at most 50, which is also the default"`
}

type cardModelListOutput struct {
	CardModels []CardModel `json:"cardModels" jsonschema:"the card models, at most 50"`
	Total      int         `json:"total" jsonschema:"the number of card models in the vault; larger than the list length when the list is truncated at the limit"`
}

func (t *Tools) addCardCardModelList(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "card_model_list",
		Title: "List card models",
		Description: "List the card models of the vault: the path of each, the name used in a card's " +
			"wikilink, the title, and the field names. Call it before writing a card, " +
			"because a card names its card model and takes the fields that card model declares. " +
			"At most 50 are returned, and `total` is the number in the vault, so a list shorter " +
			"than `total` was truncated at the limit. A card model with no fields listed could " +
			"not be read.",
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
	res := cardModelListOutput{CardModels: nil, Total: count}
	for _, s := range cardModels {
		res.CardModels = append(res.CardModels, CardModel{
			Path: s.Path, Name: domain.Basename(s.Path), Title: s.Title, Fields: s.Fields,
		})
	}
	return res, nil
}

type cardModelReadInput struct {
	Path string `json:"path" jsonschema:"the path of the card model to read, relative to the vault folder"`
}

type cardModelReadOutput struct {
	Path        string         `json:"path" jsonschema:"the path of the card model, relative to the vault folder"`
	Title       string         `json:"title" jsonschema:"the title of the card model"`
	Fields      []string       `json:"fields" jsonschema:"the field names, in order"`
	Templates   []CardTemplate `json:"templates" jsonschema:"the templates of the card model"`
	Fingerprint string         `json:"fingerprint" jsonschema:"pass to card_model_edit; a write against a changed card model is rejected"`
}

func (t *Tools) addCardCardModelRead(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "card_model_read",
		Title: "Read a card model",
		Description: "Read one card model: its title, its field names in order, and its " +
			"templates. Pass the fingerprint to `card_model_edit` to edit the card model in place.",
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
		// gomutants:disable-next-line BRANCH_IF reason="All non-Ok ReadOutcome values map to an ErrorCode in convertOutcomeToErrorCode."
		if reason, ok := convertOutcomeToErrorCode(s.Outcome); ok {
			return cardModelReadOutput{}, errors.New(describeCode(reason))
		}
		return cardModelReadOutput{}, fmt.Errorf("%s: %s", in.Path, s.Outcome)
	}
	if s.Type != format.TypeCardModel {
		return cardModelReadOutput{}, errors.New(describeCode(api.ErrorCodeNotACardModel))
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
	Path   string   `json:"path" jsonschema:"the path of the deck to read, relative to the vault folder"`
	Offset int      `json:"offset,omitempty" jsonschema:"the index of the first card to return, counted from 0"`
	Limit  int      `json:"limit,omitempty" jsonschema:"how many cards to return, at most 50"`
	CardID string   `json:"cardId,omitempty" jsonschema:"the id of one card to read alone; offset and limit are ignored when it is set"`
	Fields []string `json:"fields,omitempty" jsonschema:"the fields to return, spelled as card_read returns them; other values are left out, and a card with none of them is returned with no values"`
}

type cardReadOutput struct {
	Cards       []Card    `json:"cards" jsonschema:"the cards of the requested range, in file order"`
	Sections    []string  `json:"sections,omitempty" jsonschema:"the section names of the deck, in file order"`
	Total       int       `json:"total" jsonschema:"the number of cards in the deck"`
	Problems    []Problem `json:"problems,omitempty" jsonschema:"the problems found in the deck file"`
	Fingerprint string    `json:"fingerprint" jsonschema:"pass to a card write tool; a write against a changed deck is rejected"`
}

func (t *Tools) addCardRead(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "card_read",
		Title: "Read cards",
		Description: "Read the cards of one deck in file order, each with the card id every " +
			"other card tool takes. A deck can contain any number of cards, so this returns a range: " +
			"`total` is the number of cards, and `offset` selects the next range. Sections divide " +
			"the deck, and every card has the index of its section. Problems of the file are " +
			"returned under `problems`, on the card they concern. Pass the fingerprint to the " +
			"write tools: a write fails when the deck changed since the read.",
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
		Path: in.Path, From: in.Offset, Limit: limit, CardID: in.CardID, Fields: in.Fields,
	})
	if err != nil {
		return cardReadOutput{}, err
	}
	res := cardReadOutput{
		Cards:       nil,
		Sections:    read.Sections,
		Total:       read.Total,
		Problems:    getProblems(read.Problems),
		Fingerprint: fingerprintOf(read.FileStat),
	}
	for _, card := range read.Cards {
		res.Cards = append(res.Cards, newCard(card))
	}
	return res, nil
}
