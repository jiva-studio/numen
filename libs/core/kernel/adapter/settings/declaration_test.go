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
	holder := manifest.Manifest{ID: "md.numen.app", Name: "App", Version: "1.0.0", Settings: flat, SettingGroups: settings.GetAppSettingGroups()}
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
		"performance.profile": {"eco", "low", "balanced", "high", "maximum"},
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
		case "string":
			if len(declared.Options) == 0 {
				t.Errorf("%s has no palette choices", declared.Key)
			}
		case "boolean", "enum", "range":
		default:
			t.Errorf("%s is offered in the palette as a %s, which has no choice step", declared.Key, declared.Type)
		}
	}
}

func TestTheEditorSettingsAreDeclaredWithTheirTypeBoundsAndDefaults(t *testing.T) {
	type bounds struct {
		min, max, step float64
	}
	for key, want := range map[string]struct {
		kind         string
		defaultValue any
		bounds       *bounds
	}{
		"editor.fontSize":            {"range", 14.0, &bounds{10, 32, 1}},
		"editor.lineHeight":          {"range", 1.6, &bounds{1.2, 2.4, 0.1}},
		"editor.lineNumbers":         {"boolean", false, nil},
		"editor.highlightActiveLine": {"boolean", false, nil},
		"editor.softWrap":            {"boolean", true, nil},
		"editor.fontFamily":          {"string", "", nil},
	} {
		declared := findDeclaredSetting(t, key)
		if declared.Type != want.kind {
			t.Errorf("%s is %s, want %s", key, declared.Type, want.kind)
		}
		if declared.Default != want.defaultValue {
			t.Errorf("%s defaults to %#v, want %#v", key, declared.Default, want.defaultValue)
		}
		if declared.Title == "" {
			t.Errorf("%s has no title", key)
		}
		if want.bounds == nil {
			continue
		}
		if declared.Min == nil || *declared.Min != want.bounds.min || declared.Max == nil || *declared.Max != want.bounds.max || declared.Step == nil || *declared.Step != want.bounds.step {
			t.Errorf("%s goes %v to %v by %v, want %v", key, declared.Min, declared.Max, declared.Step, *want.bounds)
		}
	}
}

func TestTheFontFamilyOffersTheFaceOfTheThemeFirstThenSansSerifAndMono(t *testing.T) {
	declared := findDeclaredSetting(t, "editor.fontFamily")
	var values, titles []string
	for _, option := range declared.Options {
		values = append(values, option.Value)
		titles = append(titles, option.Title)
	}
	if want := []string{"", "sans", "serif", "mono"}; !slices.Equal(values, want) {
		t.Errorf("the font family offers the values %q, want %q", values, want)
	}
	if want := []string{"As in the theme", "Sans", "Serif", "Mono"}; !slices.Equal(titles, want) {
		t.Errorf("the font family offers the titles %q, want %q", titles, want)
	}
}

func TestTheEditorSettingsAreDeclaredInTheOrderOfTheEditorGroup(t *testing.T) {
	var got []string
	for _, declared := range settings.GetAppDeclaration() {
		if strings.HasPrefix(declared.Key, "editor.") {
			got = append(got, declared.Key)
		}
	}
	want := []string{"editor.fontSize", "editor.fontFamily", "editor.lineHeight", "editor.lineNumbers", "editor.highlightActiveLine", "editor.softWrap"}
	if !slices.Equal(got, want) {
		t.Errorf("the editor group declares %q, want %q", got, want)
	}
}

func TestTheEditorSettingsAreDeclaredForThePaletteWithTheirIconAndNoPreview(t *testing.T) {
	for _, key := range []string{"editor.fontSize", "editor.fontFamily", "editor.lineHeight", "editor.lineNumbers", "editor.highlightActiveLine", "editor.softWrap"} {
		declared := findDeclaredSetting(t, key)
		if !declared.Palette {
			t.Errorf("%s is not flagged for the palette", key)
		}
		if declared.Icon == "" {
			t.Errorf("%s declares no icon", key)
		}
		if declared.Preview {
			t.Errorf("%s is previewed while focused", key)
		}
	}
}

func TestTheReadingSizeKeepsItsKeyAndBoundsAndNamesBooksAndDocuments(t *testing.T) {
	declared := findDeclaredSetting(t, "appearance.textScale")
	if declared.Title != "Reading size" {
		t.Errorf("the scale of the text is titled %q", declared.Title)
	}
	if !strings.Contains(declared.Description, "book") || strings.Contains(declared.Description, "note") {
		t.Errorf("the scale of the text is described as %q, want a scale of books and documents", declared.Description)
	}
}

func TestTheShellDeclaresTheGroupsOfItsEditorAndSearchSettings(t *testing.T) {
	groups := settings.GetAppSettingGroups()
	titles := map[string]string{}
	for _, one := range groups {
		titles[one.ID] = one.Title
	}
	want := map[string]string{
		"font":        "Font",
		"display":     "Display",
		"reranking":   "Reranking",
		"results":     "Results",
		"limits":      "Query limits",
		"performance": "Performance",
	}
	if len(groups) != len(want) {
		t.Fatalf("the shell declares %d groups, want %d", len(groups), len(want))
	}
	for id, title := range want {
		if titles[id] != title {
			t.Errorf("the group %s is titled %q, want %q", id, titles[id], title)
		}
	}
}

func TestEverySettingOfTheShellNamesADeclaredGroupOrNone(t *testing.T) {
	declared := map[string]bool{}
	for _, one := range settings.GetAppSettingGroups() {
		declared[one.ID] = true
	}
	grouped := map[string]string{
		"editor.fontSize":                   "font",
		"editor.fontFamily":                 "font",
		"editor.lineHeight":                 "font",
		"editor.lineNumbers":                "display",
		"editor.highlightActiveLine":        "display",
		"editor.softWrap":                   "display",
		"search.reranking":                  "reranking",
		"search.rerankWindow":               "reranking",
		"search.maxReranked":                "reranking",
		"search.maxRerankedBytes":           "reranking",
		"search.defaultLimit":               "results",
		"search.maxLimit":                   "results",
		"search.candidatesPerCondition":     "limits",
		"search.maxCandidates":              "limits",
		"search.maxPathPrefixes":            "limits",
		"search.mergeGapBytes":              "limits",
		"search.maxQueryBytes":              "limits",
		"performance.profile":               "performance",
		"performance.disableDmabufRenderer": "performance",
	}
	for _, one := range settings.GetAppDeclaration() {
		if one.Group != "" && !declared[one.Group] {
			t.Errorf("%s names the group %q, which the shell does not declare", one.Key, one.Group)
		}
		if one.Group != grouped[one.Key] {
			t.Errorf("%s is in the group %q, want %q", one.Key, one.Group, grouped[one.Key])
		}
	}
}

func TestDmabufDefaultsToDisabledAndNamesRestart(t *testing.T) {
	declared := findDeclaredSetting(t, "performance.disableDmabufRenderer")
	if declared.Default != true {
		t.Errorf("the default is %v, want true", declared.Default)
	}
	if !strings.Contains(declared.Description, "restart") || !strings.Contains(declared.Description, "Linux") {
		t.Errorf("the description %q names no restart or no Linux", declared.Description)
	}
	if !settings.DefaultPerformance().DisableDmabufRenderer {
		t.Error("the default performance leaves the renderer on")
	}
}
