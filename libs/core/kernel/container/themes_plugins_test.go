package container

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

const (
	themesPluginID = "md.numen.themes.sample"
	nightCSS       = ":root { color-scheme: dark; --numen-surface: #101010 }"
)

func writeThemesPlugin(t *testing.T, root, id, targets string) string {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(filepath.Join(dir, "themes"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifestText := `{"id":"` + id + `","name":"Sample Themes","version":"2026.10.1","targets":[` + targets + `],
		"contributes":{"themes":[
			{"id":"night","name":"Night","appearance":"dark","file":"themes/night.css"},
			{"id":"day","name":"Day","file":"themes/day.css"}]}}`
	files := map[string]string{
		"manifest.json":    manifestText,
		"themes/night.css": nightCSS,
		"themes/day.css":   ":root { --numen-surface: #fafafa }",
	}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func openThemesConfig(t *testing.T, target string, pluginRoot string) Config {
	t.Helper()
	dir := t.TempDir()
	return NewConfig(Config{
		PluginHost: newPluginHost(t, target, pluginRoot),
		Paths:      Paths{ThemesPath: filepath.Join(dir, "themes"), SettingsPath: filepath.Join(dir, "settings.json")},
	})
}

func listThemeNames(t *testing.T, cfg Config) []string {
	t.Helper()
	service, err := cfg.Themes(nil)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := service.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, one := range answer.Msg.GetThemes() {
		names = append(names, one.GetName())
	}
	return names
}

// The theme list is read from the discovered manifests: a plugin without a process contributes its
// themes, and no plugin is started to answer.
func TestListThemesListsPluginThemesBeforeAnyPluginIsActive(t *testing.T) {
	root := t.TempDir()
	writeThemesPlugin(t, root, themesPluginID, `"desktop.editor"`)
	cfg := openThemesConfig(t, "desktop.editor", root)

	names := listThemeNames(t, cfg)

	for _, want := range []string{"preset:numen", "plugin:" + themesPluginID + "/night", "plugin:" + themesPluginID + "/day"} {
		if !slices.Contains(names, want) {
			t.Errorf("themes = %v, want %s among them", names, want)
		}
	}
	for _, info := range cfg.PluginHost.Runtime.GetPluginInfos() {
		if info.IsActive {
			t.Errorf("plugin %s is active after listing themes", info.ID)
		}
	}
}

func TestAPluginThemeIsReadFromThePluginFolder(t *testing.T) {
	root := t.TempDir()
	writeThemesPlugin(t, root, themesPluginID, `"desktop.editor"`)
	cfg := openThemesConfig(t, "desktop.editor", root)
	service, err := cfg.Themes(nil)
	if err != nil {
		t.Fatal(err)
	}

	answer, err := service.ReadTheme(t.Context(),
		connect.NewRequest(&v1.ReadThemeRequest{Name: "plugin:" + themesPluginID + "/night"}))

	if err != nil || answer.Msg.GetCss() != nightCSS {
		t.Errorf("css = %q, error = %v", answer.Msg.GetCss(), err)
	}
}

func TestListThemesListsNoThemeOfAPluginForAnotherTarget(t *testing.T) {
	root := t.TempDir()
	writeThemesPlugin(t, root, themesPluginID, `"desktop.flashcards"`)
	cfg := openThemesConfig(t, "desktop.editor", root)

	for _, name := range listThemeNames(t, cfg) {
		if strings.HasPrefix(name, "plugin:") {
			t.Errorf("the editor lists %s of a flashcards plugin", name)
		}
	}
}

func TestListThemesListsNoThemeOfADisabledPlugin(t *testing.T) {
	root := t.TempDir()
	writeThemesPlugin(t, root, themesPluginID, `"desktop.editor"`)
	cfg := openThemesConfig(t, "desktop.editor", root)
	name := "plugin:" + themesPluginID + "/night"
	service, err := cfg.Themes(nil)
	if err != nil {
		t.Fatal(err)
	}

	cfg.PluginHost.Configuration.SetSettings(map[string]map[string]any{themesPluginID: {"enabled": false}})

	if names := listThemeNames(t, cfg); slices.Contains(names, name) {
		t.Errorf("themes = %v, want no %s", names, name)
	}
	answer, err := service.ReadTheme(t.Context(), connect.NewRequest(&v1.ReadThemeRequest{Name: name}))
	if err != nil || answer.Msg.GetCss() != "" {
		t.Errorf("css = %q, error = %v, want an empty answer", answer.Msg.GetCss(), err)
	}
}

func TestAStoredThemeOfADisabledPluginFallsBackToTheDefault(t *testing.T) {
	root := t.TempDir()
	writeThemesPlugin(t, root, themesPluginID, `"desktop.editor"`)
	cfg := openThemesConfig(t, "desktop.editor", root)
	name := "plugin:" + themesPluginID + "/night"
	var said []string
	service, err := cfg.Themes(func(text string) {
		if text != "" {
			said = append(said, text)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	written, err := service.WriteAppearance(t.Context(),
		connect.NewRequest(&v1.WriteAppearanceRequest{Name: name, Mode: v1.Mode_MODE_SYSTEM}))
	if err != nil || written.Msg.GetError() != "" {
		t.Fatalf("written: %v %v", written, err)
	}
	cfg.PluginHost.Configuration.SetSettings(map[string]map[string]any{themesPluginID: {"enabled": false}})

	answer, err := service.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))

	if err != nil || answer.Msg.GetApplied() != "preset:numen" {
		t.Errorf("applied = %q, error = %v", answer.Msg.GetApplied(), err)
	}
	if len(said) == 0 || !strings.Contains(said[len(said)-1], name) {
		t.Errorf("said %v", said)
	}
}

func TestAPluginThemeLinkedOutOfThePluginFolderIsNotRead(t *testing.T) {
	root := t.TempDir()
	dir := writeThemesPlugin(t, root, themesPluginID, `"desktop.editor"`)
	cfg := openThemesConfig(t, "desktop.editor", root)
	service, err := cfg.Themes(nil)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.css")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "themes", "night.css")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("no links on this filesystem: %v", err)
	}

	answer, err := service.ReadTheme(t.Context(),
		connect.NewRequest(&v1.ReadThemeRequest{Name: "plugin:" + themesPluginID + "/night"}))

	if err == nil && strings.Contains(answer.Msg.GetCss(), "secret") {
		t.Error("a file outside the plugin folder was read")
	}
}

// A config with no plugin host offers the default theme alone.
func TestListThemesOfAConfigWithoutAPluginHostListsTheDefaultAlone(t *testing.T) {
	dir := t.TempDir()
	cfg := NewConfig(Config{
		Paths: Paths{ThemesPath: filepath.Join(dir, "themes"), SettingsPath: filepath.Join(dir, "settings.json")},
	})

	names := listThemeNames(t, cfg)

	if !slices.Equal(names, []string{"preset:numen"}) {
		t.Errorf("themes = %v", names)
	}
}
