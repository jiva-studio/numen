package service

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"

	embeddings "github.com/jiva-studio/numen/plugins/engines-embeddings"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

// Each shipped profile fetches the files that its option in the manifest names, and each of them is a
// model the manifest declares.
func TestEveryShippedProfileFetchesTheModelsItsOptionNames(t *testing.T) {
	var manifest struct {
		Dependencies []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"dependencies"`
		Settings []struct {
			Key     string `json:"key"`
			Options []struct {
				Value        string   `json:"value"`
				Dependencies []string `json:"dependencies"`
			} `json:"options"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(embeddings.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	declared := map[string]bool{}
	for _, dependency := range manifest.Dependencies {
		declared[dependency.ID] = dependency.Kind == "model"
	}
	options := map[string][]string{}
	for _, setting := range manifest.Settings {
		if setting.Key == "model" {
			for _, option := range setting.Options {
				options[option.Value] = option.Dependencies
			}
		}
	}
	table, err := model.GetShippedProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range table.GetCurated() {
		ids := slices.Sorted(maps.Values(p.Dependencies))
		if got := slices.Sorted(slices.Values(options[p.ID])); !slices.Equal(ids, got) {
			t.Errorf("%s fetches %v, its option names %v", p.ID, ids, got)
		}
		for _, id := range ids {
			if !declared[id] {
				t.Errorf("%s fetches %s, which the manifest does not declare as a model", p.ID, id)
			}
		}
	}
}
