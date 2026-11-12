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

func newReading() scenarios {
	return newScenarios(&readingPlugins{}, &readingPlugins{}, getTypes(), nil, nil)
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

	got, err := commands.Run.FindHandler(t.Context(), getVault(), handlers, "", "talks/a.mp3")

	if err != nil || got.PluginID != "org.example.ears" {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestFindHandlerReturnsTheHandlerOfThePluginGiven(t *testing.T) {
	commands := newReading()
	handlers := commands.List.ListSharedHandlers("app:file.createText")

	got, err := commands.Run.FindHandler(t.Context(), getVault(), handlers, "org.example.scans", "scans/a.pdf")
	if err != nil || got.PluginID != "org.example.scans" {
		t.Errorf("got %+v, %v", got, err)
	}
	if _, err := commands.Run.FindHandler(t.Context(), getVault(), handlers, "org.example.gone", "scans/a.pdf"); !errors.Is(err, plugincommand.ErrUnknownCommand) {
		t.Errorf("a plugin that handles nothing: %v, want ErrUnknownCommand", err)
	}
}

func TestFindHandlerNamesThePluginsWhereSeveralHold(t *testing.T) {
	commands := newReading()
	handlers := commands.List.ListSharedHandlers("app:file.createText")

	_, err := commands.Run.FindHandler(t.Context(), getVault(), handlers, "", "scans/a.pdf")

	var several *plugincommand.SeveralHandlersError
	if !errors.As(err, &several) || !slices.Equal(several.PluginIDs, []string{"org.example.layers", "org.example.scans"}) {
		t.Errorf("got %v, want the two plugins over PDFs", err)
	}
	if _, err := commands.Run.FindHandler(t.Context(), getVault(), handlers, "", "notes/a.md"); !errors.Is(err, plugincommand.ErrBadArguments) {
		t.Errorf("no handler over the file: %v, want ErrBadArguments", err)
	}
}

func TestRunPassesTheVersionArgumentToACommandWithItems(t *testing.T) {
	declared := &declaredCommands{entries: []plugin.CommandEntry{{PluginID: "org.example.proofs", Command: manifest.Command{
		ID: "proofread", Title: "Proofread", When: "version.isLive",
	}}}}
	run := newScenarios(declared, declared, getTypes(), nil, nil).Run

	_, err := run.Execute(t.Context(), getVault(), plugin.CommandCall{
		PluginID: "org.example.proofs", Command: "proofread", Path: "talks/a.mp3",
		Arguments: map[string]string{manifest.CommandArgumentVersion: "org.example.ears/ears"},
	})

	if err != nil || len(declared.calls) != 1 || declared.calls[0].Arguments[manifest.CommandArgumentVersion] != "org.example.ears/ears" {
		t.Errorf("got %v, ran %+v", err, declared.calls)
	}
}
