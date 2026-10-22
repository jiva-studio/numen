package service

import (
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/service/cardtools"
)

// Tools provides the MCP tools for interacting with flashcards and decks.
func (b *Backend) Tools() (*tools.Registry, error) {
	return cardtools.New(b.vault, b.scenarios().cards, b.scenarios().deckedit).Registry()
}
