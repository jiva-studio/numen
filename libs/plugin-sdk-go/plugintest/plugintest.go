// Package plugintest holds the checks the tests of plugins share: the dependencies a manifest declares
// and the configuration a plugin process accepts.
package plugintest

import (
	"encoding/json"
	"regexp"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// CheckDependencies fails where the manifest declares a runtime, lacks one of the dependency IDs, or,
// when isChecksumRequired, lacks the source URL or the SHA-256 checksum of a model dependency.
func CheckDependencies(t *testing.T, manifestJSON []byte, isChecksumRequired bool, ids ...string) {
	t.Helper()
	var manifest struct {
		Runtimes     []string `json:"runtimes"`
		Dependencies []struct {
			ID             string `json:"id"`
			Kind           string `json:"kind"`
			SourceURL      string `json:"sourceUrl"`
			ChecksumSHA256 string `json:"checksumSha256"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Runtimes) != 0 {
		t.Errorf("runtimes = %v, want none", manifest.Runtimes)
	}
	var declared []string
	for _, dep := range manifest.Dependencies {
		declared = append(declared, dep.ID)
		if isChecksumRequired && dep.Kind == "model" {
			if dep.SourceURL == "" {
				t.Errorf("model dependency %s has empty sourceUrl", dep.ID)
			}
			if !sha256Hex.MatchString(dep.ChecksumSHA256) {
				t.Errorf("model dependency %s has invalid sha256: %s", dep.ID, dep.ChecksumSHA256)
			}
		}
	}
	for _, expected := range ids {
		if !slices.Contains(declared, expected) {
			t.Errorf("dependencies missing %s: got %v", expected, declared)
		}
	}
}

// CheckConfigure fails where a plugin process does not accept the configuration key and keep its
// value.
func CheckConfigure(t *testing.T, key, value string) {
	t.Helper()
	life := server.NewLifecycleServer(make(chan struct{}))

	success, err := life.ApplyConfig(t.Context(), map[string]string{key: value})
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if !success {
		t.Errorf("Configure success = false, want true")
	}

	got, ok := life.GetConfig(key)
	if !ok || got != value {
		t.Errorf("%s = %q, want %q", key, got, value)
	}
}
