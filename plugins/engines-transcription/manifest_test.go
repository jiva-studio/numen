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
		Runtimes     []string `json:"runtimes"`
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
	if len(manifest.Runtimes) != 0 {
		t.Errorf("runtimes = %v, want none", manifest.Runtimes)
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
		"onnxruntime", "parakeet-encoder", "parakeet-decoder", "parakeet-joiner", "parakeet-tokens", "silero-vad",
	} {
		if !slices.Contains(ids, expected) {
			t.Errorf("dependencies missing %s: got %v", expected, ids)
		}
	}
}
