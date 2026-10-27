package cardtools

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards/deckedit"
)

var errNoVault = errors.New("no vault is open")

// The keys of what the host tells a tool about the window the agent was started from. The review
// writes the card keys, under this plugin's id, as the context of the question.
const (
	askedVault = "vault"
	askedDeck  = "md.numen.views.flashcards.studio/deck"
	askedMark  = "md.numen.views.flashcards.studio/card"
	askedFace  = "md.numen.views.flashcards.studio/face"
)

// Tools provides MCP tools for interacting with flashcards.
type Tools struct {
	vault    host.Vault
	cards    cards.Scenarios
	deckedit deckedit.Scenarios
}

// New creates a Tools instance.
func New(vault host.Vault, cards cards.Scenarios, deckedit deckedit.Scenarios) *Tools {
	return &Tools{vault: vault, cards: cards, deckedit: deckedit}
}

// Registry builds the tools registry.
func (t *Tools) Registry() (*tools.Registry, error) {
	registry := tools.New()
	for _, add := range []func(*tools.Registry) error{
		t.addCardCurrent,
		t.addCardStencilList,
		t.addCardStencilRead,
		t.addCardRead,
		t.addCardAdd,
		t.addCardEdit,
		t.addCardValueRemove,
		t.addCardRemove,
		t.addCardSectionAdd,
		t.addCardSectionRename,
		t.addCardSectionRemove,
		t.addCardDeckCreate,
		t.addCardStencilCreate,
		t.addCardStencilEdit,
		t.addCardFieldRename,
	} {
		if err := add(registry); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// getToolVault is the vault a tool works: the one the host says the agent works, else the one the
// window shows.
func (t *Tools) getToolVault(ctx context.Context) (domain.Vault, error) {
	if id := tools.Asked(ctx)[askedVault]; id != "" {
		v, err := t.vault.With(domain.VaultID(id)).GetVault(ctx)
		if errors.Is(err, host.ErrUnknownVault) {
			return domain.Vault{}, errors.New("the vault the agent works is not one this installation knows")
		}
		return v, err
	}
	v, err := t.vault.GetVault(ctx)
	if errors.Is(err, host.ErrNoVault) {
		return domain.Vault{}, errNoVault
	}
	return v, err
}
