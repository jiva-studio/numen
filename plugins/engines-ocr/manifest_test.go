package ocr

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The plugin declares the ONNX Runtime as a runtime dependency and model dependencies.
func TestManifestDeclaresDependencies(t *testing.T) {
	var manifest struct {
		Dependencies []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, dep := range manifest.Dependencies {
		ids = append(ids, dep.ID)
	}
	for _, expected := range []string{"onnxruntime", "pp-doclayout", "pp-ocr-det", "rec-pp-ocrv6-tiny", "rec-pp-ocrv5-eslav"} {
		if !slices.Contains(ids, expected) {
			t.Errorf("dependencies missing %s: got %v", expected, ids)
		}
	}
}

// manifestOf is the part of manifest.json the host reads to provision the plugin.
type manifestOf struct {
	Dependencies []struct {
		ID        string            `json:"id"`
		SourceURL string            `json:"sourceUrl"`
		Metadata  map[string]string `json:"metadata"`
	} `json:"dependencies"`
	Settings []struct {
		Key     string `json:"key"`
		Title   string `json:"title"`
		Default string `json:"default"`
		Palette bool   `json:"palette"`
		Icon    string `json:"icon"`
		Options []struct {
			Value        string   `json:"value"`
			Title        string   `json:"title"`
			Detail       string   `json:"detail"`
			Dependencies []string `json:"dependencies"`
		} `json:"options"`
	} `json:"settings"`
}

func readManifest(t *testing.T) manifestOf {
	t.Helper()
	var m manifestOf
	if err := json.Unmarshal(Manifest, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// getRequirementIDs is the ids the host provisions with model at value, the default for "": every
// dependency no option names, and the dependencies of the option in force.
func getRequirementIDs(t *testing.T, m manifestOf, value string) []string {
	t.Helper()
	named, active := map[string]bool{}, map[string]bool{}
	found := false
	for _, setting := range m.Settings {
		if setting.Key != "model" {
			continue
		}
		if value == "" {
			value = setting.Default
		}
		for _, option := range setting.Options {
			for _, id := range option.Dependencies {
				named[id] = true
				if option.Value == value {
					active[id] = true
				}
			}
			found = found || option.Value == value
		}
	}
	if !found {
		t.Fatalf("the manifest offers no model %q", value)
	}
	var ids []string
	for _, dep := range m.Dependencies {
		if !named[dep.ID] || active[dep.ID] {
			ids = append(ids, dep.ID)
		}
	}
	return ids
}

func getRecogniserIDs(ids []string) []string {
	var out []string
	for _, id := range ids {
		if strings.HasPrefix(id, "rec-") {
			out = append(out, id)
		}
	}
	return out
}

// unsafeNameChars is what the host replaces in an id to name its variable (provision.GetEnvName).
var unsafeNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// portableName is a variable name every shell accepts.
var portableName = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

// getEnvName is the variable the host gives a provisioned file in: its envVar, else one made of its id.
func getEnvName(id string, metadata map[string]string) string {
	if named := metadata["envVar"]; named != "" {
		return named
	}
	return "NUMEN_DEPENDENCY_" + strings.ToUpper(unsafeNameChars.ReplaceAllString(id, "_"))
}

func TestAFreshProfileRequiresTheFourRecognisersOfMultilingual(t *testing.T) {
	got := getRequirementIDs(t, readManifest(t), "")
	want := []string{
		"onnxruntime", "pp-doclayout", "pp-ocr-det",
		"rec-pp-ocrv5-latin", "rec-pp-ocrv5-cyrillic", "rec-pp-ocrv5-greek", "rec-pp-ocrv6-tiny",
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("a fresh profile provisions %v, want %v", got, want)
	}
}

func TestEveryRecogniserIsFoundByAVariableAndAFileOfItsOwn(t *testing.T) {
	m := readManifest(t)
	vars, files := map[string]string{}, map[string]string{}
	count := 0
	for _, dep := range m.Dependencies {
		if !strings.HasPrefix(dep.ID, "rec-") {
			continue
		}
		count++
		if got, want := dep.Metadata["fileName"], dep.ID+".onnx"; got != want {
			t.Errorf("%s is written as %q, want %q", dep.ID, got, want)
		}
		env := getEnvName(dep.ID, dep.Metadata)
		if !portableName.MatchString(env) {
			t.Errorf("%s is given in %q, which a shell cannot name", dep.ID, env)
		}
		if other, ok := vars[env]; ok {
			t.Errorf("%s and %s share the variable %s", other, dep.ID, env)
		}
		vars[env] = dep.ID
		if other, ok := files[dep.Metadata["fileName"]]; ok {
			t.Errorf("%s and %s share the file %s", other, dep.ID, dep.Metadata["fileName"])
		}
		files[dep.Metadata["fileName"]] = dep.ID
	}
	if count == 0 {
		t.Fatal("the manifest declares no recogniser")
	}
	if got := vars["NUMEN_DEPENDENCY_REC_PP_OCRV5_LATIN"]; got != "rec-pp-ocrv5-latin" {
		t.Errorf("NUMEN_DEPENDENCY_REC_PP_OCRV5_LATIN names %q, want rec-pp-ocrv5-latin", got)
	}
}

func TestEachOtherModelRequiresOnlyItsOwnRecogniser(t *testing.T) {
	m := readManifest(t)
	want := map[string][]string{
		"pp-ocrv6-tiny":       {"rec-pp-ocrv6-tiny"},
		"pp-ocrv6-small":      {"rec-pp-ocrv6-small"},
		"pp-ocrv5-latin":      {"rec-pp-ocrv5-latin"},
		"pp-ocrv5-eslav":      {"rec-pp-ocrv5-eslav"},
		"pp-ocrv5-cyrillic":   {"rec-pp-ocrv5-cyrillic"},
		"pp-ocrv5-devanagari": {"rec-pp-ocrv5-devanagari"},
		"pp-ocrv5-greek":      {"rec-pp-ocrv5-greek"},
		"custom":              {"rec-pp-ocrv6-tiny"},
	}
	for value, recognisers := range want {
		if got := getRecogniserIDs(getRequirementIDs(t, m, value)); !slices.Equal(got, recognisers) {
			t.Errorf("model %s provisions the recognisers %v, want %v", value, got, recognisers)
		}
	}
}

func TestTheModelSettingIsOfferedInThePaletteWithMultilingualFirst(t *testing.T) {
	for _, setting := range readManifest(t).Settings {
		if setting.Key != "model" {
			continue
		}
		if setting.Title != "Model" || !setting.Palette || setting.Icon != "scan-text" {
			t.Errorf("the model setting is titled %q, palette %v, icon %q; want Model, true, scan-text",
				setting.Title, setting.Palette, setting.Icon)
		}
		if setting.Default != "multilingual" {
			t.Errorf("the default model is %q, want multilingual", setting.Default)
		}
		if len(setting.Options) == 0 {
			t.Fatal("the model setting offers no option")
		}
		first := setting.Options[0]
		wantDetail := "Latin-script languages with their diacritics, Cyrillic-script languages, Greek, Chinese, English, 28 MB"
		wantIDs := []string{"rec-pp-ocrv5-latin", "rec-pp-ocrv5-cyrillic", "rec-pp-ocrv5-greek", "rec-pp-ocrv6-tiny"}
		if first.Value != "multilingual" || first.Title != "Multilingual" || first.Detail != wantDetail ||
			!slices.Equal(first.Dependencies, wantIDs) {
			t.Errorf("the first option is %+v, want multilingual titled Multilingual with %q and %v",
				first, wantDetail, wantIDs)
		}
		return
	}
	t.Fatal("the manifest declares no model setting")
}
