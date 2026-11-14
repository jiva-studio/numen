package proofreading

import (
	"encoding/json"
	"slices"
	"testing"
)

type manifestShape struct {
	Permissions []string          `json:"permissions"`
	Secrets     []json.RawMessage `json:"secrets"`
	Settings    []struct {
		Key        string   `json:"key"`
		Type       string   `json:"type"`
		Operations []string `json:"operations"`
		Max        *int     `json:"max"`
		When       any      `json:"when"`
		Default    any      `json:"default"`
		Options    []struct {
			Value string `json:"value"`
		} `json:"options"`
	} `json:"settings"`
}

func readManifest(t *testing.T) manifestShape {
	t.Helper()
	var shape manifestShape
	if err := json.Unmarshal(Manifest, &shape); err != nil {
		t.Fatal(err)
	}
	return shape
}

func TestManifestDeclaresOneSettingOfFiveRowsOfConnectionsToComplete(t *testing.T) {
	var found int
	for _, one := range readManifest(t).Settings {
		if one.Type != "connections" {
			continue
		}
		found++
		if one.Key != "connections" || !slices.Equal(one.Operations, []string{"llm.complete"}) || one.Max == nil || *one.Max != 5 || one.When != nil {
			t.Errorf("setting = %+v, want key connections, operations llm.complete, max 5 and no condition", one)
		}
	}
	if found != 1 {
		t.Errorf("the manifest declares %d settings of rows, want 1", found)
	}
}

func TestManifestDeclaresNoServiceChoiceNoKeyAndNoAddress(t *testing.T) {
	shape := readManifest(t)
	if len(shape.Secrets) != 0 {
		t.Errorf("the manifest declares %d secrets, want none", len(shape.Secrets))
	}
	for _, one := range shape.Settings {
		if slices.Contains([]string{"provider", "baseUrl", "model"}, one.Key) {
			t.Errorf("the manifest still declares the setting %s", one.Key)
		}
	}
}

func TestManifestAsksForNoNetworkAndNoCompletionPermission(t *testing.T) {
	want := []string{"vault:read", "vault:write"}
	if got := readManifest(t).Permissions; !slices.Equal(got, want) {
		t.Errorf("permissions = %v, want %v", got, want)
	}
}

func TestManifestDeclaresLanguageAndPromptAsStringsThatAreEmptyByDefault(t *testing.T) {
	for _, key := range []string{"language", "prompt"} {
		var found int
		for _, one := range readManifest(t).Settings {
			if one.Key != key {
				continue
			}
			found++
			if one.Type != "string" || one.Default != "" || one.When != nil {
				t.Errorf("setting %s = %+v, want type string, default empty and no condition", key, one)
			}
		}
		if found != 1 {
			t.Errorf("the manifest declares %d settings %s, want 1", found, key)
		}
	}
}

func TestManifestSuggestsLanguagesAndNoPromptText(t *testing.T) {
	for _, one := range readManifest(t).Settings {
		var values []string
		for _, option := range one.Options {
			values = append(values, option.Value)
		}
		switch one.Key {
		case "language":
			for _, want := range []string{"English", "Russian", "Sanskrit"} {
				if !slices.Contains(values, want) {
					t.Errorf("the language setting does not suggest %s", want)
				}
			}
		case "prompt":
			if len(values) != 0 {
				t.Errorf("the prompt setting suggests %v, want free text", values)
			}
		}
	}
}

func TestManifestPutsLanguageAndPromptAfterConnectionsAndBeforeTheSizes(t *testing.T) {
	var keys []string
	for _, one := range readManifest(t).Settings {
		keys = append(keys, one.Key)
	}
	at := func(key string) int { return slices.Index(keys, key) }
	if at("connections") < 0 || at("connections") >= at("language") || at("language") >= at("prompt") || at("prompt") >= at("batchSize") {
		t.Errorf("settings are in the order %v", keys)
	}
}
