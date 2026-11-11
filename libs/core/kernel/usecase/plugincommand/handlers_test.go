package plugincommand_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
)

// handlingPlugins declares a handler of a shared command of a plugin with a backend.
type handlingPlugins struct{ mixedCommands }

func (h *handlingPlugins) ListHandlers() []plugin.CommandEntry {
	return []plugin.CommandEntry{
		{PluginID: "md.numen.cards", Command: manifest.Command{
			ID: "app:file.createText", Title: "Read the deck", When: "target.contentType == 'application/pdf'",
		}},
	}
}

func TestRun_RunsTheHandlerOfASharedCommandInItsPlugin(t *testing.T) {
	plugins := &handlingPlugins{}
	commands := plugincommand.NewCommands(plugins, plugins, getTypes(), nil, nil)

	result, err := commands.Run.Execute(t.Context(), domain.Vault{}, plugin.CommandCall{
		PluginID: "md.numen.cards", Command: "app:file.createText", Path: "decks/tides.pdf",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message != "ran app:file.createText" || len(plugins.calls) != 1 || plugins.calls[0].Path != "decks/tides.pdf" {
		t.Errorf("got %+v, calls %+v", result, plugins.calls)
	}
}

func TestRun_RefusesAHandlerWhoseConditionDoesNotHold(t *testing.T) {
	plugins := &handlingPlugins{}
	commands := plugincommand.NewCommands(plugins, plugins, getTypes(), nil, nil)

	_, err := commands.Run.Execute(t.Context(), domain.Vault{}, plugin.CommandCall{
		PluginID: "md.numen.cards", Command: "app:file.createText", Path: "notes/tides.md",
	})
	if !errors.Is(err, plugincommand.ErrBadArguments) || len(plugins.calls) != 0 {
		t.Errorf("got %v, calls %+v", err, plugins.calls)
	}
}

func TestListing_IncludesTheHandlersAndTheContentTypes(t *testing.T) {
	plugins := &handlingPlugins{}
	commands := plugincommand.NewCommands(plugins, plugins, getTypes(), nil, nil)

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
