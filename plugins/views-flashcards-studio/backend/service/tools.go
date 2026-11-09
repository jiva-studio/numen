package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service/cardtools"
)

// Tools provides the MCP tools for interacting with flashcards and decks.
func (b *Backend) Tools() (*tools.Registry, error) {
	return cardtools.New(func(id domain.VaultID) host.VaultSelector { return b.vaults(id).Selector }, b.scenarios().cards, b.scenarios().deckedit).Registry()
}
