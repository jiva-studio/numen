package indexsource_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalogtest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/indexsource"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func manifestOf(entry catalog.Entry) map[string]any {
	kinds := make([]map[string]string, 0, len(entry.Contributes.TabKinds))
	for _, kind := range entry.Contributes.TabKinds {
		kinds = append(kinds, map[string]string{"kind": kind.Kind, "title": kind.Title, "icon": kind.Icon})
	}
	return map[string]any{
		"id": entry.ID, "name": entry.Name, "version": entry.Version, "description": entry.Description,
		"targets":  entry.Requirements.Targets,
		"frontend": map[string]any{},
		"contributes": map[string]any{
			"tabKinds": kinds, "feature": entry.Contributes.Feature, "category": entry.Contributes.Category,
		},
	}
}

func writeIndex(t *testing.T, path string, manifests []map[string]any) {
	t.Helper()
	data, err := json.Marshal(map[string]any{"plugins": manifests})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTheIndexSourceSatisfiesTheSourceContract(t *testing.T) {
	catalogtest.RunSourceContract(t, func(t *testing.T, entries []catalog.Entry) catalog.Source {
		t.Helper()
		path := filepath.Join(t.TempDir(), indexsource.FileName)
		manifests := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			manifests = append(manifests, manifestOf(entry))
		}
		writeIndex(t, path, manifests)
		return indexsource.New([]string{path}, nil)
	}, catalogtest.AvailableEntries())
}

func TestTheFirstIndexThatExistsIsRead(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	entries := catalogtest.AvailableEntries()
	second := filepath.Join(dir, "second.json")
	writeIndex(t, second, []map[string]any{manifestOf(entries[0])})

	listed, err := indexsource.New([]string{filepath.Join(dir, "absent.json"), second}, nil).List(t.Context())

	if err != nil || len(listed) != 1 || listed[0].ID != entries[0].ID {
		t.Fatalf("List = %+v, %v; want the plugin of the second index", listed, err)
	}
	if listed[0].State != catalog.StateAvailable || listed[0].Source != indexsource.Name || listed[0].Version != entries[0].Version {
		t.Errorf("entry = %+v, want an available entry of the index source with the version of the manifest", listed[0])
	}
}

func TestWithoutAnIndexTheManifestsOfTheFallbackFoldersAreRead(t *testing.T) {
	dir := t.TempDir()
	entries := catalogtest.AvailableEntries()
	for _, entry := range entries {
		folder := filepath.Join(dir, entry.ID)
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(manifestOf(entry))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "manifest.json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	listed, err := indexsource.New([]string{filepath.Join(dir, indexsource.FileName)}, []string{dir}).List(t.Context())

	if err != nil || len(listed) != len(entries) {
		t.Errorf("List = %+v, %v; want the %d plugins of the folders", listed, err, len(entries))
	}
}

func TestWithNeitherAnIndexNorFoldersNothingIsListed(t *testing.T) {
	listed, err := indexsource.New([]string{filepath.Join(t.TempDir(), indexsource.FileName)}, nil).List(t.Context())

	if err != nil || len(listed) != 0 {
		t.Errorf("List = %+v, %v; want nothing and no error", listed, err)
	}
}

func TestAnInvalidManifestIsReportedAndDoesNotHideTheOthers(t *testing.T) {
	path := filepath.Join(t.TempDir(), indexsource.FileName)
	entry := catalogtest.AvailableEntries()[0]
	writeIndex(t, path, []map[string]any{{"id": "org.example.broken"}, manifestOf(entry)})
	src := indexsource.New([]string{path}, nil)

	listed, err := src.List(t.Context())
	got, getErr := src.Get(t.Context(), entry.ID)

	if !errors.Is(err, manifest.ErrEmptyName) || len(listed) != 1 || listed[0].ID != entry.ID {
		t.Errorf("List = %+v, %v; want the valid plugin and ErrEmptyName for the other", listed, err)
	}
	if !errors.Is(getErr, manifest.ErrEmptyName) || got.ID != entry.ID {
		t.Errorf("Get = %+v, %v; want the valid plugin and ErrEmptyName", got, getErr)
	}
}

func TestAnIndexThatIsNotJSONIsAnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, indexsource.FileName)
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := indexsource.New([]string{path}, nil)

	listed, err := source.List(t.Context())
	_, getErr := source.Get(t.Context(), "org.example.any")

	var synErr *json.SyntaxError
	if !errors.As(err, &synErr) || len(listed) != 0 {
		t.Errorf("List = %+v, %v; want json.SyntaxError and no entries", listed, err)
	}
	if !errors.Is(getErr, catalog.ErrNotFound) {
		t.Errorf("Get error = %v, want catalog.ErrNotFound joined to the read error", getErr)
	}

	// Directory as index path causes os.ReadFile error != ErrNotExist
	dirSource := indexsource.New([]string{dir}, nil)
	if _, dirErr := dirSource.List(t.Context()); dirErr == nil {
		t.Error("expected error when index path is a directory")
	}

	// Fallback folders with multiple invalid plugins
	fallbackDir := t.TempDir()
	for _, bad := range []string{"bad1", "bad2"} {
		folder := filepath.Join(fallbackDir, bad)
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "manifest.json"), []byte(`{"id":"nodots"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fbSource := indexsource.New(nil, []string{fallbackDir})
	if _, fbErr := fbSource.List(t.Context()); !errors.Is(fbErr, manifest.ErrEmptyName) {
		t.Errorf("fallback discover error = %v, want ErrEmptyName", fbErr)
	}
}
