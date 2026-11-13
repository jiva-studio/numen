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
