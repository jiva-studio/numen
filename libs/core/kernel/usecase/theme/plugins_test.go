package theme_test

import (
	"strings"
	"testing"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	themeusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/theme"
)

func TestPluginThemesAreListedBetweenTheShippedAndThePersonsThemes(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	desk.writeTheme(t, "mine.css", ":root {}")

	names := getNames(themeusecase.NewListThemes(desk.sources).Execute())

	want := []string{themedomain.Default, "plugin:" + pluginID + "/night", "plugin:" + pluginID + "/day", "mine:mine"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("themes = %v, want %v", names, want)
	}
}

func TestAPluginThemeIsPinnedByTheAppearanceItDeclares(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})

	for _, one := range desk.getThemes().List {
		switch one.Name {
		case themedomain.PluginName(pluginID, "night"):
			if !one.IsPinned || one.Appearance != "dark" || one.PluginID != pluginID || one.Title != "Night" {
				t.Errorf("night = %+v", one)
			}
		case themedomain.PluginName(pluginID, "day"):
			if one.IsPinned || one.Appearance != "" {
				t.Errorf("day = %+v", one)
			}
		}
	}
}

func TestAPluginThemeIsReadByItsName(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})

	text, err := desk.readTheme(themedomain.PluginName(pluginID, "night"))

	if err != nil || text != nightCSS {
		t.Errorf("text = %q, error = %v", text, err)
	}
}

func TestAPluginNameThatNothingAnswersForIsReadAsNothing(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})

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
		if text, err := desk.readTheme(name); text != "" || err != nil {
			t.Errorf("%s: text = %q, error = %v, want nothing", name, text, err)
		}
		if err := desk.writeAppearance(themedomain.Appearance{ThemeName: name}); err == nil {
			t.Errorf("%s was written", name)
		}
	}
}

func TestABuildWithoutPluginsAnswersForNoPluginName(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	desk.sources.Plugins = nil

	if text, err := desk.readTheme(themedomain.PluginName(pluginID, "night")); text != "" || err != nil {
		t.Errorf("text = %q, error = %v, want nothing", text, err)
	}
	if hasTheme(desk.getThemes().List, themedomain.PluginName(pluginID, "night")) {
		t.Error("a build with no source lists plugin themes")
	}
}

func TestAStoredPluginThemeThatNoLongerResolvesFallsBackToTheDefaultAndWarns(t *testing.T) {
	name := themedomain.PluginName(pluginID, "night")
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: name})
	if applied := desk.getThemes().Applied; applied != name {
		t.Fatalf("applied %q before the plugin is disabled", applied)
	}

	desk.plugins.disable()

	if applied := desk.getThemes().Applied; applied != themedomain.Default {
		t.Errorf("applied %q after the plugin is disabled", applied)
	}
	warned := desk.appearance.warnings
	if len(warned) != 2 || warned[0] != "" || !strings.Contains(warned[1], name) {
		t.Errorf("said %v, want an empty text for the first list and then the warning", warned)
	}
}

func TestChoosingAPluginThemeWritesItsName(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: themedomain.Default})
	name := themedomain.PluginName(pluginID, "night")

	if err := desk.writeAppearance(themedomain.Appearance{ThemeName: name, Mode: themedomain.Dark}); err != nil {
		t.Fatalf("failed: %v", err)
	}

	if len(desk.appearance.saves) != 1 || desk.appearance.saves[0].ThemeName != name {
		t.Errorf("written: %+v", desk.appearance.saves)
	}
}
