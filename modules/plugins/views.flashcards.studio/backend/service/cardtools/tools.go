package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/tools"
)

// The keys of what the host tells a tool about the window the agent was started from.
const (
	askedVault = "vault"
	askedFocus = "focus"
	askedMark  = "mark"
	askedFace  = "face"
)

// Tools is the tools the plugin offers an agent: reading the stencils and the decks of the vault, and
// writing them.
func (b *Backend) Tools() (*tools.Registry, error) {
	registry := tools.New()
	for _, add := range []func(*tools.Registry) error{
		b.addCardCurrent,
		b.addCardStencilList,
		b.addCardStencilRead,
		b.addCardRead,
		b.addCardAdd,
		b.addCardEdit,
		b.addCardValueRemove,
		b.addCardRemove,
		b.addCardSectionAdd,
		b.addCardSectionRename,
		b.addCardSectionRemove,
		b.addCardDeckCreate,
		b.addCardStencilCreate,
		b.addCardStencilEdit,
		b.addCardFieldRename,
	} {
		if err := add(registry); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// getToolVault is the vault a tool works: the one the host says the agent works, else the one the
// window shows.
func (b *Backend) getToolVault(ctx context.Context) (domain.Vault, error) {
	if id := tools.Asked(ctx)[askedVault]; id != "" {
		v, err := b.vault.With(domain.VaultID(id)).GetVault(ctx)
		if errors.Is(err, host.ErrUnknownVault) {
			return domain.Vault{}, errors.New("the vault the agent works is not one this installation knows")
		}
		return v, err
	}
	v, err := b.vault.GetVault(ctx)
	if errors.Is(err, host.ErrNoVault) {
		return domain.Vault{}, errNoVault
	}
	return v, err
}
