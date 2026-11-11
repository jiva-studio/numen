package container

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/localsource"

	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/indexsource"
)

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

var registersTab = regexp.MustCompile(`\bregister\.(tabKind|reader)\b`)

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
	if checked < 8 {
		t.Errorf("only %d plugins register tabs from a frontend: the rule is checked against too little", checked)
	}
}

func TestTheGeneratedPluginIndexEqualsTheManifests(t *testing.T) {
	// The plugin index make plugins_index writes next to the manifests of the plugins.
	indexPath := filepath.Join(modulesRoot, "plugins", indexsource.FileName)
	manifests := getPluginManifests(t)
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("the plugin index is missing (%v): run make plugins_index", err)
	}

	listed, err := indexsource.New([]string{indexPath}, nil).List(t.Context())

	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != len(manifests) {
		t.Errorf("the index lists %d plugins and the tree holds %d manifests: run make plugins_index", len(listed), len(manifests))
	}
	for _, one := range manifests {
		want := catalog.NewEntry(indexsource.Name, catalog.StateAvailable, one.Manifest)
		index := slices.IndexFunc(listed, func(entry catalog.Entry) bool { return entry.ID == one.ID })
		if index < 0 {
			t.Errorf("the index lacks %s: run make plugins_index", one.ID)
			continue
		}
		if !reflect.DeepEqual(listed[index], want) {
			t.Errorf("%s: the index says %+v and the manifest %+v: run make plugins_index", one.ID, listed[index], want)
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
		"frontend":{},
		"contributes":{"tabKinds":[{"kind":"` + strings.ReplaceAll(id, ".", "-") + `","title":"Tab"}],"feature":"a feature","category":"views"}}`
	if err := os.WriteFile(filepath.Join(folder, "manifest.json"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTheCatalogListsThePluginsOnDiskWithTheirEnabledFlagAndTheAvailableOnes(t *testing.T) {
	dir := t.TempDir()
	writeCatalogPlugin(t, dir, "org.example.on", `"desktop.editor"`)
	writeCatalogPlugin(t, dir, "org.example.off", `"desktop.editor"`)
	writeCatalogPlugin(t, dir, "org.example.cards", `"desktop.flashcards"`)
	availableDir := t.TempDir()
	writeCatalogPlugin(t, availableDir, "org.example.offered", `"desktop.editor"`)
	host := newPluginHost(t, "desktop.editor", dir)
	host.Configuration.SetSettings(map[string]map[string]any{"org.example.off": {"enabled": false}})
	cfg := NewConfig(Config{
		PluginHost: host,
		PluginCatalog: catalog.Config{
			Target: "desktop.editor", InstallDir: "", CacheDir: "",
			Sources: []catalog.Source{
				localsource.New([]string{dir}, localsource.SettingsFunc(host.Runtime.IsPluginEnabled)),
				indexsource.New(nil, []string{availableDir}),
			},
		},
	})

	service := getPluginCatalog(plugins(cfg))
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
	if states["org.example.offered"] != catalog.StateAvailable {
		t.Errorf("a plugin of a source that is not on disk is %q, want available", states["org.example.offered"])
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
	cfg := NewConfig(Config{PluginHost: newPluginHost(t, "desktop.editor", t.TempDir())})

	if getPluginCatalog(plugins(cfg)) != nil {
		t.Error("a catalog was made with no target")
	}
}
