package plugincommand_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// readingPlugins declares the handlers of two plugins over PDFs and one over audio.
type readingPlugins struct{ declaredCommands }

func (readingPlugins) ListHandlers() []plugin.CommandEntry {
	return []plugin.CommandEntry{
		{PluginID: "org.example.scans", Command: manifest.Command{ID: "app:file.createText", Title: "Recognize", When: "target.contentType == 'application/pdf'"}},
		{PluginID: "org.example.layers", Command: manifest.Command{ID: "app:file.createText", Title: "Extract", When: "target.contentType == 'application/pdf'"}},
		{PluginID: "org.example.ears", Command: manifest.Command{ID: "app:file.createText", Title: "Transcribe", When: "target.contentType.startsWith('audio/')"}},
		{PluginID: "org.example.proofs", Command: manifest.Command{ID: "app:file.createText", Title: "Proofread", When: "version.isLive"}},
		{PluginID: "org.example.links", Command: manifest.Command{ID: "app:link.import", Title: "Import"}},
	}
}

func newReading() useCases {
	return newUseCases(&readingPlugins{}, &readingPlugins{}, getTypes())
}

func TestListSharedHandlersListsTheHandlersABackendRunsWithoutItemsByPluginID(t *testing.T) {
	var got []string
	for _, one := range newReading().List.ListSharedHandlers("app:file.createText") {
		got = append(got, one.PluginID)
	}

	if want := []string{"org.example.ears", "org.example.layers", "org.example.scans"}; !slices.Equal(got, want) {
		t.Errorf("handlers %v, want %v", got, want)
	}
}

func TestFindHandlerReturnsTheOneHandlerWhoseConditionIsTrue(t *testing.T) {
	commands := newReading()
	handlers := commands.List.ListSharedHandlers("app:file.createText")

	got, err := plugincommand.FindHandler(t.Context(), commands.Run, getVault(), handlers, "", "talks/a.mp3")

	if err != nil || got.PluginID != "org.example.ears" {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestFindHandlerReturnsTheHandlerOfThePluginGiven(t *testing.T) {
	commands := newReading()
	handlers := commands.List.ListSharedHandlers("app:file.createText")

	got, err := plugincommand.FindHandler(t.Context(), commands.Run, getVault(), handlers, "org.example.scans", "scans/a.pdf")
	if err != nil || got.PluginID != "org.example.scans" {
		t.Errorf("got %+v, %v", got, err)
	}
	if _, err := plugincommand.FindHandler(t.Context(), commands.Run, getVault(), handlers, "org.example.gone", "scans/a.pdf"); !errors.Is(err, plugin.ErrUnknownCommand) {
		t.Errorf("a plugin that handles nothing: %v, want ErrUnknownCommand", err)
	}
}

func TestFindHandlerNamesThePluginsWhereSeveralHold(t *testing.T) {
	commands := newReading()
	handlers := commands.List.ListSharedHandlers("app:file.createText")

	_, err := plugincommand.FindHandler(t.Context(), commands.Run, getVault(), handlers, "", "scans/a.pdf")

	var several *plugincommand.SeveralHandlersError
	if !errors.As(err, &several) || !slices.Equal(several.PluginIDs, []string{"org.example.layers", "org.example.scans"}) {
		t.Errorf("got %v, want the two plugins over PDFs", err)
	}
	if _, err := plugincommand.FindHandler(t.Context(), commands.Run, getVault(), handlers, "", "notes/a.md"); !errors.Is(err, plugin.ErrBadArguments) {
		t.Errorf("no handler over the file: %v, want ErrBadArguments", err)
	}
}
