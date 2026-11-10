package plugin_test

import (
	"encoding/json"
	"os"
	pathpkg "path"
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// targetVectors are the targets section of libs/core/protocol/when.json, which the plugin host derives
// the same values from.
type targetVectors struct {
	Targets struct {
		ContentTypes map[string]string `json:"contentTypes"`
		Cases        []struct {
			About  string `json:"about"`
			Target struct {
				Path     string `json:"path"`
				IsFolder bool   `json:"isFolder"`
			} `json:"target"`
			Note *struct {
				Type       string         `json:"type"`
				Properties map[string]any `json:"properties"`
			} `json:"note"`
			Values map[string]any `json:"values"`
		} `json:"cases"`
	} `json:"targets"`
}

func readTargetVectors(t *testing.T) targetVectors {
	t.Helper()
	text, err := os.ReadFile("../../../protocol/when.json")
	if err != nil {
		t.Fatal(err)
	}
	var out targetVectors
	if err := json.Unmarshal(text, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Targets.Cases) == 0 {
		t.Fatal("when.json has no target cases")
	}
	return out
}

func TestTheValuesOfATargetAreTheOnesOfWhenJSON(t *testing.T) {
	vectors := readTargetVectors(t)
	for _, one := range vectors.Targets.Cases {
		contentType := domain.ContentType(vectors.Targets.ContentTypes[strings.ToLower(strings.TrimPrefix(pathpkg.Ext(one.Target.Path), "."))])
		if contentType == "" {
			contentType = domain.UnknownContentType
		}
		values := plugin.NewTargetValues(one.Target.Path, one.Target.IsFolder, contentType)
		if one.Note != nil {
			values = values.WithNote(domain.NoteType(one.Note.Type), one.Note.Properties)
		}
		if got := values.Get(); !reflect.DeepEqual(got, one.Values) {
			t.Errorf("%s: %v, want %v", one.About, got, one.Values)
		}
	}
}

func TestTheNoteOfANoteIsUnknownUntilItIsRead(t *testing.T) {
	got := plugin.NewTargetValues("notes/Bees.md", false, "text/markdown").Get()

	if _, isGiven := got["target.noteType"]; isGiven {
		t.Errorf("the note type of a note not read is %v, want it left out", got["target.noteType"])
	}
	if _, isGiven := got["target.properties"]; isGiven {
		t.Errorf("the properties of a note not read are %v, want them left out", got["target.properties"])
	}
}

func TestTheVersionsAndTheConfigAreGivenWhereTheyAreRead(t *testing.T) {
	pathOnly := plugin.NewTargetValues("scans/a.png", false, "image/png").Get()
	if _, isGiven := pathOnly["target.versions"]; isGiven {
		t.Error("the versions are given before they are read")
	}
	if _, isGiven := pathOnly["config"]; isGiven {
		t.Error("the config is given before it is read")
	}

	got := plugin.NewTargetValues("scans/a.png", false, "image/png").
		WithVersions([]plugin.TargetVersion{{Kind: "recognition", PluginID: "org.example.scans", IsLive: true, IsValid: true}}).
		WithConfig(map[string]any{"app": map[string]any{}}).
		WithVersion("recognition", true).Get()

	want := []any{map[string]any{"kind": "recognition", "pluginId": "org.example.scans", "isLive": true, "isValid": true}}
	if !reflect.DeepEqual(got["target.versions"], want) || got["config"] == nil ||
		got["version.kind"] != "recognition" || got["version.isLive"] != true {
		t.Errorf("got %v", got)
	}
}
