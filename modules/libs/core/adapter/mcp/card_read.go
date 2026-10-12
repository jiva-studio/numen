package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
)

func addCardReadTools(server *sdk.Server, core Core) {
	addCardStencilList(server, core)
	addCardRead(server, core)
}

// addCardFocus adds the tool that returns the card the person is looking at.
//
// It is added only when the core has a reviewing session.
func addCardFocus(server *sdk.Server, core Core) {
	if core.GetCurrentCard == nil {
		return
	}

	addTool(server, &sdk.Tool{
		Name:  "card_current",
		Title: "Show the card in front",
		Description: "The card in front of the person: the deck it stands in, the mark " +
			"`card_read` addresses it by, and the face it is being shown through. Ask it " +
			"before saying anything about the card they are on — a person answers one " +
			"card and moves to the next while you work — and read the card itself with " +
			"`card_read`. Nothing is in front of them between cards, and the deck comes " +
			"back empty.",
	}, func(
		_ context.Context, _ *sdk.CallToolRequest, _ struct{},
	) (*sdk.CallToolResult, CurrentCard, error) {
		return nil, core.GetCurrentCard(), nil
	})
}

type stencilListInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"how many stencils to return, at most 50, which is also the default"`
}

type stencilListOutput struct {
	Stencils []Stencil `json:"stencils"`
	Total    int       `json:"total" jsonschema:"how many stencils the vault holds, which stands above the list when the ceiling was reached"`
}

func addCardStencilList(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "card_stencil_list",
		Title: "List stencils",
		Description: "The stencils of the vault: where each is filed, the name a card's " +
			"wikilink writes, what it is called, and what a card cut by it is asked for. " +
			"Call this before writing a card, because a card names its stencil and asks " +
			"for the fields that stencil declares. At most 50 come back, and `total` says " +
			"how many the vault holds, so a list shorter than that was cut by the " +
			"ceiling. A stencil with no fields listed is one that could not be read.",
	}, newHandler(func(ctx context.Context, in stencilListInput) (stencilListOutput, error) {
		return listCardStencils(ctx, core, in)
	}))
}

func listCardStencils(ctx context.Context, core Core, in stencilListInput) (stencilListOutput, error) {
	if in.Limit > maxStencils {
		return stencilListOutput{}, fmt.Errorf("ask for at most %d stencils at a time", maxStencils)
	}
	limit := maxStencils
	if in.Limit > 0 {
		limit = in.Limit
	}
	stencils, count, err := core.Cards.List.Execute(ctx, core.getCurrentVault().Vault, limit)
	if err != nil {
		return stencilListOutput{}, err
	}
	res := stencilListOutput{Total: count}
	for _, s := range stencils {
		res.Stencils = append(res.Stencils, Stencil{
			Path: s.Path, Name: domain.Basename(s.Path), Title: s.Title, Fields: s.Fields,
		})
	}
	return res, nil
}

type cardReadInput struct {
	Path   string   `json:"path" jsonschema:"the deck to read"`
	From   int      `json:"from,omitempty" jsonschema:"which card to start at, counted from zero"`
	Limit  int      `json:"limit,omitempty" jsonschema:"how many cards to return"`
	Mark   string   `json:"mark,omitempty" jsonschema:"one card's mark, to read that card alone; from and limit say nothing about a call naming it"`
	Fields []string `json:"fields,omitempty" jsonschema:"the fields to answer with, spelled as the card writes them, which is what card_read gives; every other value is left out, and a card writing none of them comes back holding nothing"`
}

type cardReadOutput struct {
	Cards       []Card   `json:"cards"`
	Sections    []string `json:"sections,omitempty" jsonschema:"what the deck's sections are called, in the order they stand in the file"`
	Total       int      `json:"total" jsonschema:"how many cards the deck holds"`
	Faults      []Fault  `json:"faults,omitempty"`
	Fingerprint string   `json:"fingerprint" jsonschema:"hand this to a writing tool to refuse a write over an edit you did not see"`
}

func addCardRead(server *sdk.Server, core Core) {
	addTool(server, &sdk.Tool{
		Name:  "card_read",
		Title: "Read cards",
		Description: "The cards of one deck, in the order they stand in the file, each " +
			"under the mark every other tool here addresses it by. A deck holds as many " +
			"cards as a person writes, so this answers a run of them at a time: `total` " +
			"says how many there are, and `from` takes the next run. The sections are " +
			"the runs a person divided the deck into, and every card says which of them " +
			"it stands under. What was wrong with the file comes back under `faults`, " +
			"on the card it is against. The fingerprint is what the writing tools want: " +
			"hand it back and a write is refused if the person changed the deck in the " +
			"meantime.",
	}, newHandler(func(ctx context.Context, in cardReadInput) (cardReadOutput, error) {
		return readCards(ctx, core, in)
	}))
}

func readCards(ctx context.Context, core Core, in cardReadInput) (cardReadOutput, error) {
	// The limit applies to the number of cards one call reads.
	if in.Mark == "" && in.Limit > maxCards {
		return cardReadOutput{}, fmt.Errorf("read at most %d cards at a time", maxCards)
	}
	deckContents, err := readDeck(ctx, core, in.Path)
	if err != nil {
		return cardReadOutput{}, err
	}
	res := cardReadOutput{
		Total:       len(deckContents.Body.Cards),
		Faults:      faults(deckContents.Body.Problems),
		Fingerprint: fingerprintOf(deckContents.Fingerprint),
	}
	for _, s := range deckContents.Body.Sections {
		res.Sections = append(res.Sections, s.Name)
	}
	if in.Mark != "" {
		at, err := index(deckContents.Body.Cards, in.Mark)
		if err != nil {
			return cardReadOutput{}, err
		}
		res.Cards = append(res.Cards, keepFields(newCard(deckContents.Body.Cards[at]), in.Fields))
		return res, nil
	}
	for _, card := range getCardRange(deckContents.Body.Cards, in.From, in.Limit) {
		res.Cards = append(res.Cards, keepFields(newCard(card), in.Fields))
	}
	return res, nil
}

// getCardRange returns at most limit cards starting at from, clamped to the deck.
// A limit of zero or less means the maximum one call may read.
func getCardRange(deck []format.Card, from, limit int) []format.Card {
	if limit <= 0 {
		limit = maxCards
	}
	from = min(max(from, 0), len(deck))
	return deck[from:min(from+limit, len(deck))]
}

// readDeck reads the deck at path in the current vault, and refuses a file
// that is not a deck.
func readDeck(ctx context.Context, core Core, path string) (cards.DeckContents, error) {
	return readDeckIn(ctx, core, core.getCurrentVault().Vault, path)
}

// readDeckIn reads the deck at path in v, and refuses a file that is not a deck.
func readDeckIn(ctx context.Context, core Core, v domain.Vault, path string) (cards.DeckContents, error) {
	contents, err := core.Cards.Read.Deck(ctx, v, path)
	if err != nil {
		return cards.DeckContents{}, err
	}
	if why := whyNotADeck(contents); why != "" {
		return cards.DeckContents{}, fmt.Errorf("%s: %s", path, why)
	}
	return contents, nil
}
