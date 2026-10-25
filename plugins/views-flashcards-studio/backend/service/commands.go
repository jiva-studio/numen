package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
)

// commandCreateDeck is the command the manifest declares for making a deck.
const commandCreateDeck = "create-deck"

// Commands runs the commands the manifest declares.
func (b *Backend) Commands() (*command.Registry, error) {
	registry := command.New()
	if err := registry.Add(commandCreateDeck, b.runCreateDeck); err != nil {
		return nil, err
	}
	return registry, nil
}

// runCreateDeck makes a deck of no cards under the title, in the folder where one is given.
func (b *Backend) runCreateDeck(ctx context.Context, request command.Request) (string, error) {
	v, err := b.vault.GetVault(ctx)
	if errors.Is(err, host.ErrNoVault) {
		return "", errNoVault
	}
	if err != nil {
		return "", err
	}
	made, err := b.scenarios().cards.Create.Deck(ctx, v, cards.New{
		Title: request.Arguments.GetString("title"),
		Path:  request.Arguments.GetString("folder"),
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Created the deck %s", made.Path), nil
}
