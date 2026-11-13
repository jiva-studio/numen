package transcription

import (
	"encoding/json"
	"regexp"
	"slices"
	"testing"
)

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// The plugin declares the ONNX Runtime and model dependencies with checksums.
func TestManifestDeclaresDependencies(t *testing.T) {
	var manifest struct {
		Dependencies []struct {
			ID             string `json:"id"`
			Kind           string `json:"kind"`
			SourceURL      string `json:"sourceUrl"`
			ChecksumSHA256 string `json:"checksumSha256"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, dep := range manifest.Dependencies {
		ids = append(ids, dep.ID)
		if dep.Kind == "model" {
			if dep.SourceURL == "" {
				t.Errorf("model dependency %s has empty sourceUrl", dep.ID)
			}
			if !sha256Hex.MatchString(dep.ChecksumSHA256) {
				t.Errorf("model dependency %s has invalid sha256: %s", dep.ID, dep.ChecksumSHA256)
			}
		}
	}
	for _, expected := range []string{
		"onnxruntime", "silero-vad",
		"parakeet-v3-encoder", "parakeet-v3-decoder", "parakeet-v3-joiner", "parakeet-v3-tokens",
		"parakeet-v2-encoder", "parakeet-v2-decoder", "parakeet-v2-joiner", "parakeet-v2-tokens",
	} {
		if !slices.Contains(ids, expected) {
			t.Errorf("dependencies missing %s: got %v", expected, ids)
		}
	}
}

// The process keeps its loaded model for ten minutes without a call.
func TestTheManifestKeepsTheProcessForTenIdleMinutes(t *testing.T) {
	var manifest struct {
		Backend struct {
			IdleStopSeconds int `json:"idleStopSeconds"`
		} `json:"backend"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}

	if manifest.Backend.IdleStopSeconds != 600 {
		t.Errorf("idleStopSeconds = %d, want 600", manifest.Backend.IdleStopSeconds)
	}
}
