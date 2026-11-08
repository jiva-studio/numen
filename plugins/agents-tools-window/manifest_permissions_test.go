package window

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestManifestDeclaresTheWindowPermission(t *testing.T) {
	var manifest struct {
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	want := []string{"vault:read", "window:focus"}
	got := slices.Clone(manifest.Permissions)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("the permissions are %v, want %v", manifest.Permissions, want)
	}
}
