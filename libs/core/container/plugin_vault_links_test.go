package container_test

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// windowIndex is the index updater of a window's note use cases, which records the paths it is given.
type windowIndex struct {
	mu    sync.Mutex
	paths []string
}

func (w *windowIndex) getPaths() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.paths)
}

// openWindowScenarios is the window's note use cases over the assembly, whose index updater records
// the paths it updates before it refreshes them.
func openWindowScenarios(assembly container.EditorAssembly) (container.PluginVaultHooks, *windowIndex) {
	refresh := assembly.GetRefresh()
	index := &windowIndex{}
	scenarios := assembly.OpenNotes(func(ctx context.Context, v domain.Vault, paths []string) error {
		index.mu.Lock()
		index.paths = append(index.paths, paths...)
		index.mu.Unlock()
		_, err := refresh.Execute(ctx, v, paths)
		return err
	})
	return container.PluginVaultHooks{Notes: scenarios}, index
}

func statFile(t *testing.T, shared pluginport.PluginVault, path string) domain.FileStat {
	t.Helper()
	stat, err := shared.StatFile(t.Context(), path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return stat
}

func TestALinkAPluginAddsInTheShownVaultIsSavedThroughTheWindowsNoteUseCases(t *testing.T) {
	notes := map[string]string{
		"Danube.md": "# Danube\n\nA river of central Europe.\n",
		"Rhine.md":  "# Rhine\n\nA river of western Europe.\n",
	}
	v, current, assembly := openPluginVault(t, notes)
	hooks, index := openWindowScenarios(assembly)
	shared, err := assembly.OpenPluginVault(current, hooks)
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Scan(t.Context(), []string{"Danube.md", "Rhine.md"}); err != nil {
		t.Fatal(err)
	}
	before := len(index.getPaths())

	added, err := shared.AddLinks(t.Context(), []pluginport.PluginLinkAddition{{
		From: "Rhine.md", Baseline: statFile(t, shared, "Rhine.md"),
		Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Danube"}, Role: domain.RoleRef},
	}})
	if err != nil {
		t.Fatalf("the addition failed: %v", err)
	}

	if len(added) != 1 || added[0].Err != nil {
		t.Fatalf("the addition answered %+v, want one entry with no error", added)
	}
	if got := index.getPaths()[before:]; !slices.Contains(got, "Rhine.md") {
		t.Errorf("the window's index updater was given %v, want Rhine.md", got)
	}
	if content, _ := readVaultFile(t, v, "Rhine.md"); !strings.Contains(content, "to: Danube") || !strings.Contains(content, "role: ref") {
		t.Errorf("Rhine.md does not hold the link to Danube:\n%s", content)
	}
	if targets := getLinkTargets(t, shared, "Rhine.md"); !slices.Contains(targets, "Danube.md") {
		t.Errorf("the links of Rhine.md reach %v, want Danube.md", targets)
	}
}

func TestALinkAPluginRemovesInTheShownVaultIsSavedThroughTheWindowsNoteUseCases(t *testing.T) {
	v, current, assembly := openPluginVault(t, getRivers())
	hooks, index := openWindowScenarios(assembly)
	shared, err := assembly.OpenPluginVault(current, hooks)
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Scan(t.Context(), []string{"Danube.md", "Deltas.md", "Estuaries.md"}); err != nil {
		t.Fatal(err)
	}
	before := len(index.getPaths())

	if _, err := shared.RemoveLinks(t.Context(), "Deltas.md", domain.Address{Scheme: domain.SchemeName, Value: "Danube"}, "",
		statFile(t, shared, "Deltas.md")); err != nil {
		t.Fatalf("the removal failed: %v", err)
	}

	if got := index.getPaths()[before:]; !slices.Contains(got, "Deltas.md") {
		t.Errorf("the window's index updater was given %v, want Deltas.md", got)
	}
	if content, _ := readVaultFile(t, v, "Deltas.md"); strings.Contains(content, "Danube") {
		t.Errorf("Deltas.md still links to Danube:\n%s", content)
	}
}

func TestALinkAPluginAddsInAnotherVaultIsNotSavedThroughTheWindowsNoteUseCases(t *testing.T) {
	other, current, assembly := openTwoPluginVaults(t)
	hooks, index := openWindowScenarios(assembly)
	shared, err := assembly.OpenPluginVault(current, hooks)
	if err != nil {
		t.Fatal(err)
	}
	lakes, err := shared.With(other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := lakes.Scan(t.Context(), []string{"Baikal.md", "Lakes.md"}); err != nil {
		t.Fatal(err)
	}
	before := len(index.getPaths())

	added, err := lakes.AddLinks(t.Context(), []pluginport.PluginLinkAddition{{
		From: "Lakes.md", Baseline: statFile(t, lakes, "Lakes.md"),
		Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Baikal"}, Role: domain.RoleChild},
	}})
	if err != nil || len(added) != 1 || added[0].Err != nil {
		t.Fatalf("the addition in %s answered %+v, %v, want one entry with no error", other.Name, added, err)
	}

	if got := index.getPaths()[before:]; len(got) != 0 {
		t.Errorf("the window showing another vault had its index updater given %v from %s", got, other.Name)
	}
	if content, _ := readVaultFile(t, other, "Lakes.md"); !strings.Contains(content, "to: Baikal") {
		t.Errorf("Lakes.md does not hold the link to Baikal:\n%s", content)
	}
}
