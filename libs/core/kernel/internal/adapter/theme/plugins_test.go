package theme_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/theme"
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
			{PluginID: pluginID, ID: "day", Name: "Day"},
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

func TestPluginThemesAreListedBetweenTheShippedAndThePersonsThemes(t *testing.T) {
	catalog := createThemeFolder(t).WithPlugins(createFakePlugins())
	writeTheme(t, catalog, "mine.css", ":root {}")

	names := getNames(catalog.Themes())

	want := []string{theme.Default, "plugin:" + pluginID + "/night", "plugin:" + pluginID + "/day", "mine:mine"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("themes = %v, want %v", names, want)
	}
}

func TestAPluginThemeIsPinnedByTheAppearanceItDeclares(t *testing.T) {
	catalog := createThemeFolder(t).WithPlugins(createFakePlugins())

	for _, one := range catalog.Themes() {
		switch one.Name {
		case theme.PluginName(pluginID, "night"):
			if !one.IsPinned || one.Appearance != "dark" || one.PluginID != pluginID || one.Title != "Night" {
				t.Errorf("night = %+v", one)
			}
		case theme.PluginName(pluginID, "day"):
			if one.IsPinned || one.Appearance != "" {
				t.Errorf("day = %+v", one)
			}
		}
	}
}

func TestAPluginThemeIsReadByItsName(t *testing.T) {
	catalog := createThemeFolder(t).WithPlugins(createFakePlugins())

	text, err := catalog.Text(theme.PluginName(pluginID, "night"))

	if err != nil || text != nightCSS {
		t.Errorf("text = %q, error = %v", text, err)
	}
}

func TestAPluginNameThatNothingAnswersForIsNoTheme(t *testing.T) {
	catalog := createThemeFolder(t).WithPlugins(createFakePlugins())

	for _, name := range []string{
		"plugin:",
		"plugin:" + pluginID,
		"plugin:" + pluginID + "/",
		"plugin:/night",
		"plugin:" + pluginID + "/gone",
		"plugin:org.example.other/night",
		"plugin:" + pluginID + "/../night",
		"preset:dracula",
	} {
		if text, err := catalog.Text(name); !errors.Is(err, theme.ErrNoTheme) {
			t.Errorf("%s: text = %q, error = %v, want ErrNoTheme", name, text, err)
		}
	}
}

func TestACatalogueWithoutPluginsAnswersForNoPluginName(t *testing.T) {
	catalog := createThemeFolder(t)

	if _, err := catalog.Text(theme.PluginName(pluginID, "night")); !errors.Is(err, theme.ErrNoTheme) {
		t.Errorf("error = %v, want ErrNoTheme", err)
	}
	if catalog.HasPlugins() {
		t.Error("a catalog with no source reports plugins")
	}
}

func TestAStoredPluginThemeThatNoLongerResolvesFallsBackToTheDefaultAndWarns(t *testing.T) {
	plugins := createFakePlugins()
	name := theme.PluginName(pluginID, "night")
	fakeSettings := createFakeAppearances(t, theme.Appearance{ThemeName: name})
	fakeSettings.service.Catalog = fakeSettings.service.Catalog.WithPlugins(plugins)
	if applied := fakeSettings.themes(t).GetApplied(); applied != name {
		t.Fatalf("applied %q before the plugin is disabled", applied)
	}

	plugins.disable()

	if applied := fakeSettings.themes(t).GetApplied(); applied != theme.Default {
		t.Errorf("applied %q after the plugin is disabled", applied)
	}
	if len(fakeSettings.reply) != 2 || fakeSettings.reply[0] != "" || !strings.Contains(fakeSettings.reply[1], name) {
		t.Errorf("said %v, want an empty text for the first list and then the warning", fakeSettings.reply)
	}
}

func TestListThemesCarriesThePluginAndTheAppearanceOfAPluginTheme(t *testing.T) {
	fakeSettings := createFakeAppearances(t, theme.Appearance{ThemeName: theme.Default})
	fakeSettings.service.Catalog = fakeSettings.service.Catalog.WithPlugins(createFakePlugins())

	found := map[string]*v1.Theme{}
	for _, one := range fakeSettings.themes(t).GetThemes() {
		found[one.GetName()] = one
	}

	night := found[theme.PluginName(pluginID, "night")]
	if night.GetOrigin() != v1.Origin_ORIGIN_PLUGIN || night.GetPluginId() != pluginID ||
		night.GetAppearance() != "dark" || !night.GetIsPinned() || night.GetTitle() != "Night" {
		t.Errorf("night = %v", night)
	}
	day := found[theme.PluginName(pluginID, "day")]
	if day.GetAppearance() != "" || day.GetIsPinned() {
		t.Errorf("day = %v", day)
	}
}

func TestChoosingAPluginThemeWritesItsName(t *testing.T) {
	fakeSettings := createFakeAppearances(t, theme.Appearance{ThemeName: theme.Default})
	fakeSettings.service.Catalog = fakeSettings.service.Catalog.WithPlugins(createFakePlugins())
	name := theme.PluginName(pluginID, "night")

	if failed := fakeSettings.choose(t, name, v1.Mode_MODE_DARK); failed != "" {
		t.Fatalf("failed: %s", failed)
	}

	if len(fakeSettings.content) != 1 || fakeSettings.content[0].ThemeName != name {
		t.Errorf("written: %+v", fakeSettings.content)
	}
}

func TestReadThemeAnswersWithTheStylesheetOfAPluginTheme(t *testing.T) {
	fakeSettings := createFakeAppearances(t, theme.Appearance{ThemeName: theme.Default})
	fakeSettings.service.Catalog = fakeSettings.service.Catalog.WithPlugins(createFakePlugins())

	answer, err := fakeSettings.service.ReadTheme(t.Context(),
		connect.NewRequest(&v1.ReadThemeRequest{Name: theme.PluginName(pluginID, "night")}))

	if err != nil || answer.Msg.GetCss() != nightCSS {
		t.Errorf("css = %q, error = %v", answer.Msg.GetCss(), err)
	}
}

func TestEveryOriginOfAThemeIsNamedByItsPrefix(t *testing.T) {
	for origin, prefix := range map[theme.Origin]string{
		theme.Preset: "preset:", theme.Mine: "mine:", theme.Plugin: "plugin:",
	} {
		if string(origin)+":" != prefix {
			t.Errorf("origin %q is named with %q", origin, prefix)
		}
	}
}

func openWatch(t *testing.T, service *theme.Service) *connect.ServerStreamForClient[v1.WatchThemesResponse] {
	t.Helper()
	server := serveThemes(t, service)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	client := numenv1connect.NewThemeServiceClient(server.Client(), server.URL)
	stream, err := client.WatchThemes(ctx, connect.NewRequest(&v1.WatchThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	})
	return stream
}

func serveThemes(t *testing.T, service *theme.Service) *httptest.Server {
	t.Helper()
	route, handler := numenv1connect.NewThemeServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func readReported(t *testing.T, stream *connect.ServerStreamForClient[v1.WatchThemesResponse], want ...string) {
	t.Helper()
	reported := map[string]bool{}
	for stream.Receive() {
		for _, name := range stream.Msg().GetNames() {
			reported[name] = true
		}
		if slices.ContainsFunc(want, func(name string) bool { return !reported[name] }) {
			continue
		}
		return
	}
	t.Fatalf("the stream ended with %v reported, want %v: %v", reported, want, stream.Err())
}

// WatchThemes reports the plugin themes that vanish with the plugin that is disabled.
func TestWatchThemesReportsThePluginThemesOfAPluginDisabled(t *testing.T) {
	plugins := createFakePlugins()
	service := createFakeAppearances(t, theme.Appearance{}).service
	service.Catalog = service.Catalog.WithPlugins(plugins)
	stream := openWatch(t, service)

	plugins.disable()

	readReported(t, stream, theme.PluginName(pluginID, "night"), theme.PluginName(pluginID, "day"))
}

// WatchThemes reports the plugin themes that appear with the plugin that is enabled.
func TestWatchThemesReportsThePluginThemesOfAPluginEnabled(t *testing.T) {
	plugins := createFakePlugins()
	plugins.disable()
	service := createFakeAppearances(t, theme.Appearance{}).service
	service.Catalog = service.Catalog.WithPlugins(plugins)
	stream := openWatch(t, service)

	plugins.enable(createFakePlugins().themes...)

	readReported(t, stream, theme.PluginName(pluginID, "night"), theme.PluginName(pluginID, "day"))
}

// A catalog with no folder but with plugins still streams.
func TestWatchThemesOfACatalogueWithoutAFolderStillWatchesPlugins(t *testing.T) {
	plugins := createFakePlugins()
	service := &theme.Service{Catalog: theme.Catalog{}.WithPlugins(plugins)}
	stream := openWatch(t, service)

	plugins.disable()

	readReported(t, stream, theme.PluginName(pluginID, "night"))
}
