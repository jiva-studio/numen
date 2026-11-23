package plugincommand_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// handlingPlugins declares a handler of a shared command of a plugin with a backend.

// handlingPlugins declares a handler of a shared command of a plugin with a backend.
type handlingPlugins struct{ mixedCommands }

func (h *handlingPlugins) ListHandlers() []plugin.CommandEntry {
	return []plugin.CommandEntry{
		{PluginID: "md.numen.cards", Command: manifest.Command{
			ID: "app:file.createText", Title: "Read the deck", When: "target.contentType == 'application/pdf'",
		}},
	}
}

func TestListing_IncludesTheHandlersAndTheContentTypes(t *testing.T) {
	plugins := &handlingPlugins{}
	commands := newUseCases(plugins, plugins, getTypes())

	listing := commands.Listing.Execute()

	if len(listing.Handlers) != 1 || listing.Handlers[0].PluginID != "md.numen.cards" || listing.Handlers[0].HasItems {
		t.Errorf("handlers %+v, want the one of md.numen.cards without items", listing.Handlers)
	}
	if len(listing.ContentTypeByExtension) == 0 {
		t.Error("the listing holds no content type")
	}
	if len(listing.Commands) != len(commands.List.ExecuteAll()) {
		t.Errorf("%d commands listed, want %d", len(listing.Commands), len(commands.List.ExecuteAll()))
	}
}
