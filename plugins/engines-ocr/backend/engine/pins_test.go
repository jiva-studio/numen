package engine

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"testing"

	ocr "github.com/jiva-studio/numen/plugins/engines-ocr"
)

// manifestModels is the model setting and the dependencies of manifest.json.
type manifestModels struct {
	Dependencies []struct {
		ID        string            `json:"id"`
		SourceURL string            `json:"sourceUrl"`
		Metadata  map[string]string `json:"metadata"`
	} `json:"dependencies"`
	Settings []struct {
		Key     string `json:"key"`
		Options []struct {
			Value        string   `json:"value"`
			Dependencies []string `json:"dependencies"`
		} `json:"options"`
	} `json:"settings"`
}

// hostNameChars is what the host replaces in an id to name its variable (provision.GetEnvName).
var hostNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]`)

func getIDs(files []RecogniserFile) []string {
	ids := make([]string, 0, len(files))
	for _, file := range files {
		ids = append(ids, file.ID)
	}
	return ids
}

func TestFindRecognisersNamesTheFilesTheManifestProvisions(t *testing.T) {
	var m manifestModels
	if err := json.Unmarshal(ocr.Manifest, &m); err != nil {
		t.Fatal(err)
	}
	type provisioned struct{ address, env, fileName string }
	deps := map[string]provisioned{}
	for _, dep := range m.Dependencies {
		env := dep.Metadata["envVar"]
		if env == "" {
			env = "NUMEN_DEPENDENCY_" + strings.ToUpper(hostNameChars.ReplaceAllString(dep.ID, "_"))
		}
		deps[dep.ID] = provisioned{address: dep.SourceURL, env: env, fileName: dep.Metadata["fileName"]}
	}
	count := 0
	for _, setting := range m.Settings {
		if setting.Key != "model" {
			continue
		}
		for _, option := range setting.Options {
			count++
			files, ok := FindRecognisers(option.Value)
			if !ok {
				t.Errorf("the manifest offers %s, which the engine does not run", option.Value)
				continue
			}
			want := option.Dependencies
			if option.Value == RecogniserCustom {
				want = nil
			}
			if got := getIDs(files); !slices.Equal(got, want) {
				t.Errorf("%s reads with %v, want %v", option.Value, got, want)
			}
			for _, file := range files {
				dep := deps[file.ID]
				if file.Address != dep.address || file.EnvVar != dep.env || file.FileName != dep.fileName {
					t.Errorf("%s: the engine looks for %+v, the host provisions %+v", option.Value, file, dep)
				}
			}
		}
	}
	if count == 0 {
		t.Fatal("the manifest offers no model")
	}
}

func TestFindRecognisersListsMultilingualInTieBreakOrder(t *testing.T) {
	files, ok := FindRecognisers(RecogniserMultilingual)
	want := []string{"rec-pp-ocrv5-latin", "rec-pp-ocrv5-cyrillic", "rec-pp-ocrv5-greek", "rec-pp-ocrv6-tiny"}
	if got := getIDs(files); !ok || !slices.Equal(got, want) {
		t.Errorf("multilingual reads with %v (%v), want %v", got, ok, want)
	}
}

func TestFindRecognisersNamesNoFileForTheCustomModel(t *testing.T) {
	if files, ok := FindRecognisers(RecogniserCustom); !ok || len(files) != 0 {
		t.Errorf("the custom model reads with %v (%v), want no file", files, ok)
	}
}

func TestFindRecognisersRefusesAModelThePluginDoesNotProvision(t *testing.T) {
	if files, ok := FindRecognisers("tesseract"); ok {
		t.Errorf("an unknown model reads with %v", files)
	}
}
