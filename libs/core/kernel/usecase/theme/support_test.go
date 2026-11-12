package theme_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/themefiles"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/watchhub"
	themeusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/theme"
)

const (
	pluginID = "org.example.themes"
	nightCSS = ":root { --numen-surface: #101010 }"
)

// fakePlugins is a source of plugin themes held in memory.
type fakePlugins struct {
	mu     sync.Mutex
	themes []themedomain.PluginTheme
	texts  map[string]string
}

func createFakePlugins() *fakePlugins {
	return &fakePlugins{
		themes: []themedomain.PluginTheme{
			{PluginID: pluginID, ID: "night", Name: "Night", Appearance: "dark"},
			{PluginID: pluginID, ID: "day", Name: "Day", Appearance: ""},
		},
		texts: map[string]string{
			pluginID + "/night": nightCSS,
			pluginID + "/day":   ":root { --numen-surface: #fafafa }",
		},
	}
}

func (f *fakePlugins) ListThemes() []themedomain.PluginTheme {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]themedomain.PluginTheme(nil), f.themes...)
}

func (f *fakePlugins) ReadTheme(pluginID, themeID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, one := range f.themes {
		if one.PluginID == pluginID && one.ID == themeID {
			return f.texts[pluginID+"/"+themeID], nil
		}
	}
	return "", themedomain.ErrNoTheme
}

func (f *fakePlugins) enable(themes ...themedomain.PluginTheme) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.themes = append(f.themes, themes...)
}

func (f *fakePlugins) disable() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.themes = nil
}

// fakeAppearances is the settings behind the themes, held in memory.
type fakeAppearances struct {
	appearance themedomain.Appearance
	saves      []themedomain.Appearance
	warnings   []string
	failure    error
}

func (d *fakeAppearances) ReadAppearance() (themedomain.Appearance, error) {
	return d.appearance, nil
}

func (d *fakeAppearances) WriteAppearance(one themedomain.Appearance) error {
	if d.failure != nil {
		return d.failure
	}
	d.saves = append(d.saves, one)
	d.appearance = one
	return nil
}

func (d *fakeAppearances) WarnTheme(text string) { d.warnings = append(d.warnings, text) }

// themesDesk is the use cases over a real themes folder and the presets, a fake plugin source and
// fake settings.
type themesDesk struct {
	files      themefiles.Files
	plugins    *fakePlugins
	appearance *fakeAppearances
	sources    themeusecase.Sources
	scales     *themeusecase.Scales
}

func createThemesDesk(t *testing.T, stored themedomain.Appearance) *themesDesk {
	t.Helper()
	files, err := themefiles.OpenAt(filepath.Join(t.TempDir(), "numen", "themes"), watchhub.NewHub())
	if err != nil {
		t.Fatal(err)
	}
	plugins := createFakePlugins()
	return &themesDesk{
		files:      files,
		plugins:    plugins,
		appearance: &fakeAppearances{appearance: stored, saves: nil, warnings: nil, failure: nil},
		sources:    themeusecase.Sources{Presets: files, Own: files, Plugins: plugins},
		scales:     themeusecase.NewScales(0, 0),
	}
}

func (d *themesDesk) getThemes() themeusecase.Themes {
	return themeusecase.NewGetThemes(d.sources, d.appearance, d.appearance, d.scales,
		themeusecase.Bounds{Min: 0.8, Max: 2}, themeusecase.Bounds{Min: 0.8, Max: 1.75}).Execute()
}

func (d *themesDesk) writeAppearance(choice themedomain.Appearance) error {
	return themeusecase.NewWriteAppearance(d.sources, d.appearance, d.appearance, d.scales).Execute(choice)
}

func (d *themesDesk) readTheme(name string) (string, error) {
	return themeusecase.NewReadTheme(d.sources).Execute(name)
}

func (d *themesDesk) writeTheme(t *testing.T, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(d.files.Dir(), name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func getNames(themes []themedomain.Theme) []string {
	names := make([]string, 0, len(themes))
	for _, one := range themes {
		names = append(names, one.Name)
	}
	return names
}

func hasTheme(themes []themedomain.Theme, name string) bool {
	for _, one := range themes {
		if one.Name == name {
			return true
		}
	}
	return false
}
