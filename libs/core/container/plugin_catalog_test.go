package container

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/hintsource"
)

// readFrontendSource is the text of the sources of a plugin's frontend that are not tests.
func readFrontendSource(t *testing.T, dir string) string {
	t.Helper()
	var all strings.Builder
	for _, path := range listFrontendSources(t, filepath.Join(dir, "frontend")) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		all.Write(data)
	}
	return all.String()
}

var registersTab = regexp.MustCompile(`\bregister(TabKind|Reader)\b`)

func TestAPluginThatRegistersTabsDeclaresThemInItsManifest(t *testing.T) {
	checked := 0
	for _, one := range getPluginManifests(t) {
		source := readFrontendSource(t, one.Dir)
		if !registersTab.MatchString(source) {
			continue
		}
		checked++
		if len(one.Contributes.TabKinds) == 0 {
			t.Errorf("%s registers tabs from its frontend and declares no contributes.tabKinds", one.Dir)
		}
		for _, kind := range one.Contributes.TabKinds {
			literal := strings.TrimPrefix(kind.Kind, "reader:")
			if !strings.Contains(source, "'"+literal+"'") && !strings.Contains(source, `"`+literal+`"`) {
				t.Errorf("%s declares the tab kind %q, which its frontend does not register", one.Dir, kind.Kind)
			}
		}
	}
	if checked < 9 {
		t.Errorf("only %d plugins register tabs from a frontend: the rule is checked against too little", checked)
	}
}

func TestEveryHintRepeatsTheManifestOfThePluginItNames(t *testing.T) {
	manifests := getPluginManifests(t)
	hinted := map[string]bool{}
	for _, hint := range hintsource.BuiltIn() {
		hinted[hint.ID] = true
		index := slices.IndexFunc(manifests, func(one pluginManifest) bool { return one.ID == hint.ID })
		if index < 0 {
			t.Errorf("the hint %s names a plugin that has no manifest", hint.ID)
			continue
		}
		declared := manifests[index]
		if hint.Name != declared.Name || hint.Description != declared.Description {
			t.Errorf("%s: the hint says %q, %q and the manifest %q, %q", hint.ID, hint.Name, hint.Description, declared.Name, declared.Description)
		}
		if hint.Contributes.Feature != declared.Contributes.Feature || hint.Contributes.Category != declared.Contributes.Category ||
			!slices.Equal(hint.Contributes.TabKinds, declared.Contributes.TabKinds) {
			t.Errorf("%s: the hint contributes %+v and the manifest %+v", hint.ID, hint.Contributes, declared.Contributes)
		}
	}
	for _, one := range manifests {
		if len(one.Contributes.TabKinds) > 0 && !hinted[one.ID] {
			t.Errorf("%s declares tab kinds and has no hint, so a tab of its kind has no placeholder before the plugin is installed", one.ID)
		}
	}
}

func writeCatalogPlugin(t *testing.T, dir, id, targets string) {
	t.Helper()
	folder := filepath.Join(dir, id)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	text := `{"id":"` + id + `","name":"` + id + `","version":"1.0.0","targets":[` + targets + `],
		"frontend":{"entrypoint":"dist/index.js"},
		"contributes":{"tabKinds":[{"kind":"` + strings.ReplaceAll(id, ".", "-") + `","title":"Tab"}],"feature":"a feature","category":"views"}}`
	if err := os.WriteFile(filepath.Join(folder, "manifest.json"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTheCatalogListsThePluginsOnDiskWithTheirEnabledFlagAndTheHints(t *testing.T) {
	dir := t.TempDir()
	writeCatalogPlugin(t, dir, "org.example.on", `"desktop.editor"`)
	writeCatalogPlugin(t, dir, "org.example.off", `"desktop.editor"`)
	writeCatalogPlugin(t, dir, "org.example.cards", `"desktop.flashcards"`)
	host := NewPluginHost("desktop.editor", dir)
	host.SetSettings(map[string]map[string]any{"org.example.off": {"enabled": false}})
	cfg := Config{
		PluginHost:    host,
		PluginDirs:    []string{dir},
		PluginCatalog: catalog.Config{Target: "desktop.editor", AppVersion: "2026.10.0", Sources: nil, InstallDir: "", CacheDir: ""},
	}

	service := cfg.GetPluginCatalog()
	if service == nil {
		t.Fatal("no catalog")
	}
	listing, err := service.List(t.Context())

	if err != nil {
		t.Fatalf("List: %v", err)
	}
	states := map[string]catalog.State{}
	for _, one := range listing.Entries {
		states[one.ID] = one.State
	}
	if states["org.example.on"] != catalog.StateInstalled || states["org.example.off"] != catalog.StateDisabled {
		t.Errorf("states = %v, want the enabled plugin installed and the other disabled", states)
	}
	if states["md.numen.readers.pdf"] != catalog.StateAvailable {
		t.Errorf("a plugin that ships with the application and is not on disk is %q, want available", states["md.numen.readers.pdf"])
	}
	cards, ok := findEntry(listing, "org.example.cards")
	if !ok || cards.Incompatibility == nil {
		t.Errorf("a plugin of another window is %+v, want it listed with the reason it cannot run here", cards)
	}
	found, findErr := service.FindByTabKind(t.Context(), "org-example-off")
	if findErr != nil || found.ID != "org.example.off" {
		t.Errorf("FindByTabKind = %+v, %v; want the disabled plugin", found, findErr)
	}
}

func findEntry(listing catalog.Listing, id string) (catalog.Entry, bool) {
	for _, one := range listing.Entries {
		if one.ID == id {
			return one, true
		}
	}
	return catalog.Entry{}, false
}

func TestAnInstallationThatNamesNoTargetHasNoCatalog(t *testing.T) {
	cfg := Config{PluginHost: NewPluginHost("desktop.editor", t.TempDir())}

	if cfg.GetPluginCatalog() != nil {
		t.Error("a catalog was made with no target")
	}
	if cfg.getEditorCatalog() != nil {
		t.Error("the editor was handed a catalog with no target")
	}
}
