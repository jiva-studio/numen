package window

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestManifestDeclaresTheWindowPermissionAndTheNextVersion(t *testing.T) {
	var manifest struct {
		Version     string   `json:"version"`
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != "2026.11.5" {
		t.Errorf("the version is %q, want 2026.11.5", manifest.Version)
	}
	want := []string{"vault:read", "window:focus"}
	got := slices.Clone(manifest.Permissions)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("the permissions are %v, want %v", manifest.Permissions, want)
	}
}
