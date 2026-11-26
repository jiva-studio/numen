package service

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"

	reranking "github.com/jiva-studio/numen/plugins/engines-reranking"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
)

// Each file a shipped profile fetches is a model the manifest declares, and each model the manifest
// declares is a file of a shipped profile.
func TestEveryShippedProfileFetchesModelsTheManifestDeclares(t *testing.T) {
	var manifest struct {
		Dependencies []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(reranking.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	var declared []string
	for _, dependency := range manifest.Dependencies {
		if dependency.Kind == "model" {
			declared = append(declared, dependency.ID)
		}
	}
	table, err := profile.GetShippedProfiles()
	if err != nil {
		t.Fatal(err)
	}
	var fetched []string
	for _, p := range table.GetCurated() {
		fetched = append(fetched, slices.Collect(maps.Values(p.Dependencies))...)
	}
	if !slices.Equal(slices.Sorted(slices.Values(fetched)), slices.Sorted(slices.Values(declared))) {
		t.Errorf("the profiles fetch %v, the manifest declares %v", fetched, declared)
	}
}
