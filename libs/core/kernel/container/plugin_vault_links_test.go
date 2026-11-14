package container_test

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"

	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/note/domain"
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

func openWindowUseCases(assembly windowVaults) (container.PluginVaultHooks, *windowIndex) {
	refresh := assembly.GetRefresh()
	index := &windowIndex{}
	useCases := assembly.OpenNotes(func(ctx context.Context, v domain.Vault, paths []string) error {
		index.mu.Lock()
		index.paths = append(index.paths, paths...)
		index.mu.Unlock()
		_, err := refresh.Execute(ctx, v, paths)
		return err
	})
	return container.PluginVaultHooks{Notes: &useCases}, index
}

func statFile(t *testing.T, shared *vaultapi.Access, path string) domain.FileStat {
	t.Helper()
	stat, err := shared.Files.StatFile(t.Context(), path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return stat
}

func TestPluginLinkAddInActiveVaultUsesWindowNoteUseCases(t *testing.T) {
	notes := map[string]string{
		"Danube.md": "# Danube\n\nA river of central Europe.\n",
		"Rhine.md":  "# Rhine\n\nA river of western Europe.\n",
	}
	v, current, assembly := openPluginVault(t, notes)
	hooks, index := openWindowUseCases(assembly)
	shared, err := assembly.OpenPluginVault(current, hooks)
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Writer.ScanFiles(t.Context(), []string{"Danube.md", "Rhine.md"}); err != nil {
		t.Fatal(err)
	}
	before := len(index.getPaths())

	added, err := shared.LinkEditor.AddLinks(t.Context(), []plugin.LinkAddition{{
		From: "Rhine.md", Baseline: statFile(t, shared, "Rhine.md"),
		Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Danube"}, Role: domain.RoleRef},
	}})
	if err != nil {
		t.Fatalf("the addition failed: %v", err)
	}

	if len(added) != 1 || added[0].Err != nil {
		t.Fatalf("the addition returned %+v, want one entry with no error", added)
	}
	if got := index.getPaths()[before:]; !slices.Contains(got, "Rhine.md") {
		t.Errorf("the window's index updater was given %v, want Rhine.md", got)
	}
	if content, _ := readVaultFile(t, v, "Rhine.md"); !strings.Contains(content, "to: Danube") || !strings.Contains(content, "role: ref") {
		t.Errorf("Rhine.md does not contain the link to Danube:\n%s", content)
	}
	if targets := getLinkTargets(t, shared, "Rhine.md"); !slices.Contains(targets, "Danube.md") {
		t.Errorf("the links of Rhine.md point to %v, want Danube.md", targets)
	}
}

func TestPluginLinkRemoveInActiveVaultUsesWindowNoteUseCases(t *testing.T) {
	v, current, assembly := openPluginVault(t, getRivers())
	hooks, index := openWindowUseCases(assembly)
	shared, err := assembly.OpenPluginVault(current, hooks)
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Writer.ScanFiles(t.Context(), []string{"Danube.md", "Deltas.md", "Estuaries.md"}); err != nil {
		t.Fatal(err)
	}
	before := len(index.getPaths())

	if _, err := shared.LinkEditor.RemoveLinks(t.Context(), "Deltas.md", domain.Address{Scheme: domain.SchemeName, Value: "Danube"}, "",
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

func TestPluginLinkAddInOtherVaultBypassesWindowNoteUseCases(t *testing.T) {
	other, current, assembly := openTwoPluginVaults(t)
	hooks, index := openWindowUseCases(assembly)
	shared, err := assembly.OpenPluginVault(current, hooks)
	if err != nil {
		t.Fatal(err)
	}
	lakes, err := shared.Switch.With(other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := lakes.Writer.ScanFiles(t.Context(), []string{"Baikal.md", "Lakes.md"}); err != nil {
		t.Fatal(err)
	}
	before := len(index.getPaths())

	added, err := lakes.LinkEditor.AddLinks(t.Context(), []plugin.LinkAddition{{
		From: "Lakes.md", Baseline: statFile(t, lakes, "Lakes.md"),
		Link: domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Baikal"}, Role: domain.RoleChild},
	}})
	if err != nil || len(added) != 1 || added[0].Err != nil {
		t.Fatalf("the addition in %s returned %+v, %v, want one entry with no error", other.Name, added, err)
	}

	if got := index.getPaths()[before:]; len(got) != 0 {
		t.Errorf("the window showing another vault had its index updater given %v from %s", got, other.Name)
	}
	if content, _ := readVaultFile(t, other, "Lakes.md"); !strings.Contains(content, "to: Baikal") {
		t.Errorf("Lakes.md does not contain the link to Baikal:\n%s", content)
	}
}
