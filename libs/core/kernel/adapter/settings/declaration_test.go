package settings_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	domainsettings "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func findDeclaredSetting(t *testing.T, key string) manifest.Setting {
	t.Helper()
	for _, one := range settings.GetAppDeclaration() {
		if one.Key == key {
			return one
		}
	}
	t.Fatalf("the shell does not declare %s", key)
	return manifest.Setting{}
}

func TestTheShellDeclaresItsSixSettingsInTheFormatOfAManifest(t *testing.T) {
	for _, key := range []string{
		"appearance.theme",
		"appearance.mode",
		"appearance.interfaceScale",
		"appearance.textScale",
		"naming.syncTitleAndFilename",
		"performance.profile",
	} {
		declared := findDeclaredSetting(t, key)
		if declared.Title == "" {
			t.Errorf("%s has no title", key)
		}
	}
}

func TestTheDeclarationOfTheShellBreaksNoRuleOfAManifest(t *testing.T) {
	declared := settings.GetAppDeclaration()
	if len(declared) == 0 {
		t.Fatal("the shell declares nothing")
	}
	// The key of a shell setting is a path joined by dots; a manifest key of a plugin is one segment.
	flat := make([]manifest.Setting, 0, len(declared))
	for _, one := range declared {
		one.Key = strings.ReplaceAll(one.Key, ".", "")
		flat = append(flat, one)
	}
	holder := manifest.Manifest{ID: "md.numen.app", Name: "App", Version: "1.0.0", Settings: flat}
	if issues := holder.ListIssues(); len(issues) != 0 {
		t.Errorf("the declaration breaks %v", issues)
	}
}

func TestTheScalesAreRangesWithTheirBoundsAndDefaults(t *testing.T) {
	for key, bounds := range map[string][2]float64{
		"appearance.interfaceScale": {0.8, 2},
		"appearance.textScale":      {0.8, 1.75},
	} {
		declared := findDeclaredSetting(t, key)
		if declared.Type != "range" {
			t.Errorf("%s is %s", key, declared.Type)
		}
		if declared.Min == nil || *declared.Min != bounds[0] || declared.Max == nil || *declared.Max != bounds[1] {
			t.Errorf("%s goes from %v to %v, want %v", key, declared.Min, declared.Max, bounds)
		}
		if declared.Default != 1.0 {
			t.Errorf("%s defaults to %v", key, declared.Default)
		}
	}
}

func TestTheModeAndTheProfileAreEnumsOfTheWordsTheFileUses(t *testing.T) {
	for key, want := range map[string][]string{
		"appearance.mode":     {"system", "light", "dark"},
		"performance.profile": {"eco", "balanced", "maximum"},
	} {
		declared := findDeclaredSetting(t, key)
		var got []string
		for _, option := range declared.Options {
			got = append(got, option.Value)
		}
		slices.Sort(got)
		slices.Sort(want)
		if declared.Type != "enum" || !slices.Equal(got, want) {
			t.Errorf("%s is %s of %v, want an enum of %v", key, declared.Type, got, want)
		}
	}
	if got := findDeclaredSetting(t, "appearance.mode").Default; got != "system" {
		t.Errorf("the mode defaults to %v", got)
	}
	if got := findDeclaredSetting(t, "performance.profile").Default; got != "balanced" {
		t.Errorf("the profile defaults to %v", got)
	}
}

func TestTheNamingIsABooleanOnByDefault(t *testing.T) {
	declared := findDeclaredSetting(t, "naming.syncTitleAndFilename")
	if declared.Type != "boolean" || declared.Default != true {
		t.Errorf("the naming is %s defaulting to %v", declared.Type, declared.Default)
	}
}

func TestTheThemeListsItsOptionsFromTheRegistryOfInstalledThemes(t *testing.T) {
	declared := findDeclaredSetting(t, "appearance.theme")
	if declared.OptionsFrom != "themes" {
		t.Errorf("the theme lists its options from %q", declared.OptionsFrom)
	}
	if len(declared.Options) != 0 {
		t.Errorf("the theme lists options of its own: %v", declared.Options)
	}
	if declared.Default != settings.DefaultTheme {
		t.Errorf("the theme defaults to %v", declared.Default)
	}
}

func TestTheSettingsThatApplyWhileFocusedCarryThePreviewFlag(t *testing.T) {
	if !findDeclaredSetting(t, "appearance.theme").Preview {
		t.Error("the theme is not previewed while focused")
	}
	if findDeclaredSetting(t, "naming.syncTitleAndFilename").Preview {
		t.Error("the naming is previewed while focused")
	}
}

func TestEveryDefaultOfTheShellIsReadFromTheDeclaration(t *testing.T) {
	file := settings.DefaultFile()
	for _, declared := range settings.GetAppDeclaration() {
		if declared.Default == nil {
			continue
		}
		path := append([]string{domainsettings.AppOwner}, strings.Split(declared.Key, ".")...)
		got, ok := file.GetValue(path...)
		if !ok || got != declared.Default {
			t.Errorf("%v is %v (found %v) in the default file, and %v in the declaration", path, got, ok, declared.Default)
		}
	}
}

func TestTheSixSettingsOfTheShellAreDeclaredForThePaletteWithTheirIcon(t *testing.T) {
	for _, key := range []string{
		"appearance.theme",
		"appearance.mode",
		"appearance.interfaceScale",
		"appearance.textScale",
		"naming.syncTitleAndFilename",
		"performance.profile",
	} {
		declared := findDeclaredSetting(t, key)
		if !declared.Palette {
			t.Errorf("%s is not flagged for the palette", key)
		}
		if declared.Icon == "" {
			t.Errorf("%s declares no icon", key)
		}
	}
}

func TestEverySettingFlaggedForThePaletteIsABooleanAnEnumOrARange(t *testing.T) {
	for _, declared := range settings.GetAppDeclaration() {
		if !declared.Palette {
			continue
		}
		switch declared.Type {
		case "boolean", "enum", "range":
		default:
			t.Errorf("%s is offered in the palette as a %s, which has no choice step", declared.Key, declared.Type)
		}
	}
}
