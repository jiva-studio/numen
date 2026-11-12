// Package cardtools provides the MCP tools that read and edit card models, decks and cards.
package cardtools

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards/deckedit"
)

var errNoVault = errors.New("the call names no vault")

// The keys of the context the host passes to a tool about the window the agent
// was started from. The review writes them, under this plugin's id, as the
// context of the request.
const (
	requestedDeck     = "md.numen.views.flashcards.studio/deck"
	requestedCard     = "md.numen.views.flashcards.studio/card"
	requestedTemplate = "md.numen.views.flashcards.studio/template"
)

// Tools provides MCP tools for interacting with flashcards.
type Tools struct {
	vaults       func(domain.VaultID) host.VaultSelector
	cards        cards.Scenarios
	bounded      cards.RenameField
	deckCards    deckedit.CardScenarios
	deckSections deckedit.SectionScenarios
}

// New creates a Tools instance. The rename an agent runs is bounded; the one in cards is not.
func New(
	vaults func(domain.VaultID) host.VaultSelector, cards cards.Scenarios, bounded cards.RenameField,
	deckCards deckedit.CardScenarios, deckSections deckedit.SectionScenarios,
) *Tools {
	return &Tools{vaults: vaults, cards: cards, bounded: bounded, deckCards: deckCards, deckSections: deckSections}
}

// Registry builds the tools registry.
func (t *Tools) Registry() (*tools.Registry, error) {
	registry := tools.New()
	for _, add := range []func(*tools.Registry) error{
		t.addCardCurrentGet,
		t.addCardCardModelList,
		t.addCardCardModelRead,
		t.addCardRead,
		t.addCardAdd,
		t.addCardEdit,
		t.addCardValueRemove,
		t.addCardRemove,
		t.addCardSectionAdd,
		t.addCardSectionRename,
		t.addCardSectionRemove,
		t.addCardDeckCreate,
		t.addCardCardModelCreate,
		t.addCardCardModelEdit,
		t.addCardFieldRename,
	} {
		// gomutants:disable-next-line BRANCH_IF reason="Static tool specifications with distinct names always register on a new registry."
		if err := add(registry); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (t *Tools) getToolVault(ctx context.Context) (domain.Vault, error) {
	id := tools.GetVault(ctx)
	if id == "" {
		return domain.Vault{}, errNoVault
	}
	v, err := t.vaults(domain.VaultID(id)).GetVault(ctx)
	if errors.Is(err, host.ErrUnknownVault) {
		return domain.Vault{}, errors.New("the agent vault is not registered in this installation")
	}
	return v, err
}
