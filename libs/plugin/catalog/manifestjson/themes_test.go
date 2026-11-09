package manifestjson_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const themesOnlyHead = `{"id":"org.example.themes","name":"Themes","version":"1.0.0","targets":["desktop.editor"]`

func TestAManifestOfThemesOnlyNeedsNeitherBackendNorFrontend(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(themesOnlyHead + `,"contributes":{"themes":[
		{"id":"night","name":"Night","appearance":"dark","file":"themes/night.css"},
		{"id":"day","name":"Day","file":"day.css"}]}}`))
	if err != nil {
		t.Fatal(err)
	}

	if m.HasBackend() || m.HasFrontend() {
		t.Fatalf("the manifest declares a runnable part: %+v %+v", m.Backend, m.Frontend)
	}
	want := []manifest.Theme{
		{ID: "night", Name: "Night", Appearance: "dark", File: "themes/night.css"},
		{ID: "day", Name: "Day", Appearance: "", File: "day.css"},
	}
	if len(m.Contributes.Themes) != len(want) {
		t.Fatalf("themes = %+v", m.Contributes.Themes)
	}
	for i, theme := range want {
		if m.Contributes.Themes[i] != theme {
			t.Errorf("theme %d = %+v, want %+v", i, m.Contributes.Themes[i], theme)
		}
	}
}

func TestAThemeThatTheApplicationCannotUseIsRejected(t *testing.T) {
	cases := map[string]string{
		"an id in capitals":     `{"id":"Night","name":"N","file":"n.css"}`,
		"an empty id":           `{"id":"","name":"N","file":"n.css"}`,
		"an id declared twice":  `{"id":"n","name":"N","file":"n.css"},{"id":"n","name":"M","file":"m.css"}`,
		"no name":               `{"id":"n","name":" ","file":"n.css"}`,
		"an unknown appearance": `{"id":"n","name":"N","appearance":"sepia","file":"n.css"}`,
		"no file":               `{"id":"n","name":"N"}`,
		"a parent path":         `{"id":"n","name":"N","file":"../n.css"}`,
		"a nested parent path":  `{"id":"n","name":"N","file":"a/../../n.css"}`,
		"an absolute path":      `{"id":"n","name":"N","file":"/etc/n.css"}`,
		"a backslash path":      `{"id":"n","name":"N","file":"a\\n.css"}`,
		"an unclean path":       `{"id":"n","name":"N","file":"a//n.css"}`,
		"a file that is no css": `{"id":"n","name":"N","file":"n.txt"}`,
		"a bare extension":      `{"id":"n","name":"N","file":".css"}`,
	}
	for name, themes := range cases {
		_, err := manifestjson.Read(strings.NewReader(themesOnlyHead + `,"contributes":{"themes":[` + themes + `]}}`))

		if !errors.Is(err, manifest.ErrInvalidContribution) {
			t.Errorf("%s: error = %v, want an invalid contribution", name, err)
		}
	}
}

func writeThemesPlugin(t *testing.T, root, themes string) string {
	t.Helper()
	dir := filepath.Join(root, "themes-plugin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := themesOnlyHead + `,"contributes":{"themes":[` + themes + `]}}`
	if err := os.WriteFile(filepath.Join(dir, manifest.FileName), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDiscoveryLoadsAPluginOfThemesOnly(t *testing.T) {
	root := t.TempDir()
	dir := writeThemesPlugin(t, root, `{"id":"night","name":"Night","file":"themes/night.css"}`)
	if err := os.Mkdir(filepath.Join(dir, "themes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "themes", "night.css"), []byte(":root{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, problems := manifestjson.Discover([]string{root})

	if len(problems) != 0 || len(found) != 1 || found[0].Manifest.ID != "org.example.themes" {
		t.Fatalf("found = %+v, problems = %+v", found, problems)
	}
}

func TestDiscoveryReportsAThemeFileThatCannotBeRead(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside.css")
	if err := os.WriteFile(outside, []byte(":root{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(dir string) error{
		"a missing file": func(string) error { return nil },
		"a directory": func(dir string) error {
			return os.Mkdir(filepath.Join(dir, "n.css"), 0o755)
		},
		"a link out of the plugin folder": func(dir string) error {
			return os.Symlink(outside, filepath.Join(dir, "n.css"))
		},
		"a link to the parent folder": func(dir string) error {
			return os.Symlink("..", filepath.Join(dir, "n.css"))
		},
		"an oversized file": func(dir string) error {
			return os.WriteFile(filepath.Join(dir, "n.css"), []byte(strings.Repeat("a", manifest.MaxThemeSize+1)), 0o644)
		},
	}
	for name, prepare := range cases {
		root := t.TempDir()
		dir := writeThemesPlugin(t, root, `{"id":"n","name":"N","file":"n.css"}`)
		if err := prepare(dir); err != nil {
			t.Fatal(err)
		}

		found, problems := manifestjson.Discover([]string{root})

		if len(found) != 0 || len(problems) != 1 {
			t.Errorf("%s: found = %+v, problems = %+v", name, found, problems)
		}
		if name == "a missing file" && !errors.Is(problems[0].Err, os.ErrNotExist) {
			t.Errorf("%s: expected os.ErrNotExist, got %v", name, problems[0].Err)
		}
		if name == "a link to the parent folder" && !strings.Contains(problems[0].Err.Error(), "leaves the plugin folder") {
			t.Errorf("%s: expected leaves the plugin folder, got %v", name, problems[0].Err)
		}
	}
}

func TestAThemeFileLinkedInsideThePluginFolderIsRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "real", "n.css"), []byte("a{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("real", "n.css"), filepath.Join(dir, "n.css")); err != nil {
		t.Fatal(err)
	}

	text, err := manifestjson.ReadThemeFile(dir, "n.css")

	if err != nil || text != "a{}" {
		t.Errorf("text = %q, error = %v", text, err)
	}

	// Exact MaxThemeSize boundary is accepted.
	maxContent := strings.Repeat("a", manifest.MaxThemeSize)
	if err := os.WriteFile(filepath.Join(dir, "max.css"), []byte(maxContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := manifestjson.ReadThemeFile(dir, "max.css"); err != nil || len(got) != manifest.MaxThemeSize {
		t.Errorf("ReadThemeFile max.css: len=%d, err=%v", len(got), err)
	}
	if _, err := manifestjson.ReadThemeFile(dir, "../bad.css"); err == nil {
		t.Error("expected error for invalid theme file name")
	}
	if _, err := manifestjson.ReadThemeFile(filepath.Join(dir, "nonexistent"), "n.css"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected ErrNotExist for nonexistent dir, got %v", err)
	}
}
