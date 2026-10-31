package localsource_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/catalogtest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/localsource"
)

// settings answers from the set of ids that are switched off.
type settings struct{ disabled map[string]bool }

func (s settings) IsEnabled(pluginID string) bool { return !s.disabled[pluginID] }

func writePlugin(t *testing.T, dir, id string, manifest map[string]any) {
	t.Helper()
	folder := filepath.Join(dir, id)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "manifest.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func manifestOf(entry catalog.Entry) map[string]any {
	kinds := make([]map[string]string, 0, len(entry.Contributes.TabKinds))
	for _, kind := range entry.Contributes.TabKinds {
		kinds = append(kinds, map[string]string{"kind": kind.Kind, "title": kind.Title, "icon": kind.Icon})
	}
	return map[string]any{
		"id": entry.ID, "name": entry.Name, "version": entry.Version, "description": entry.Description,
		"targets":  entry.Requirements.Targets,
		"frontend": map[string]string{"entrypoint": "dist/index.js"},
		"contributes": map[string]any{
			"tabKinds": kinds, "feature": entry.Contributes.Feature, "category": entry.Contributes.Category,
		},
	}
}

func TestTheLocalSourceSatisfiesTheSourceContract(t *testing.T) {
	disabledEntries := catalogtest.OnDiskEntries()
	catalogtest.RunSourceContract(t, func(t *testing.T, entries []catalog.Entry) catalog.Source {
		dir := t.TempDir()
		off := map[string]bool{}
		for _, entry := range entries {
			writePlugin(t, dir, entry.ID, manifestOf(entry))
			off[entry.ID] = entry.State == catalog.StateDisabled
		}
		return localsource.New([]string{dir}, settings{disabled: off})
	}, disabledEntries)
}

func TestADisabledPluginIsListedWithItsState(t *testing.T) {
	dir := t.TempDir()
	for _, entry := range catalogtest.OnDiskEntries() {
		writePlugin(t, dir, entry.ID, manifestOf(entry))
	}
	source := localsource.New([]string{dir}, settings{disabled: map[string]bool{"org.example.beta": true}})

	entries, err := source.List(t.Context())

	if err != nil || len(entries) != 2 {
		t.Fatalf("List = %v, %v", entries, err)
	}
	states := map[string]catalog.State{}
	for _, entry := range entries {
		states[entry.ID] = entry.State
	}
	if states["org.example.alpha"] != catalog.StateInstalled || states["org.example.beta"] != catalog.StateDisabled {
		t.Errorf("states = %v", states)
	}
}

func TestAPluginOfAnotherWindowIsListedAndKeepsItsTargets(t *testing.T) {
	dir := t.TempDir()
	entry := catalogtest.OnDiskEntries()[0]
	entry.Requirements.Targets = []string{"desktop.flashcards"}
	writePlugin(t, dir, entry.ID, manifestOf(entry))

	entries, err := localsource.New([]string{dir}, settings{disabled: nil}).List(t.Context())

	if err != nil || len(entries) != 1 || entries[0].Requirements.Targets[0] != "desktop.flashcards" {
		t.Errorf("List = %+v, %v", entries, err)
	}
}

func TestABrokenFolderIsReportedWithThePluginsThatRead(t *testing.T) {
	dir := t.TempDir()
	entry := catalogtest.OnDiskEntries()[0]
	writePlugin(t, dir, entry.ID, manifestOf(entry))
	writePlugin(t, dir, "broken", map[string]any{"id": "nodots", "name": "Broken", "version": "1.0.0"})

	entries, err := localsource.New([]string{dir}, settings{disabled: nil}).List(t.Context())

	if len(entries) != 1 || entries[0].ID != entry.ID {
		t.Errorf("entries = %+v, want the valid plugin", entries)
	}
	if err == nil || errors.Is(err, catalog.ErrNotFound) {
		t.Errorf("error = %v, want the broken folder reported", err)
	}
}

func TestAFolderThatDoesNotExistListsNothing(t *testing.T) {
	source := localsource.New([]string{filepath.Join(t.TempDir(), "absent")}, settings{disabled: nil})

	entries, err := source.List(t.Context())

	if err != nil || len(entries) != 0 {
		t.Errorf("List = %v, %v", entries, err)
	}
}
