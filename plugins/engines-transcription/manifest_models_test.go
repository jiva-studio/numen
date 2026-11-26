package transcription

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

type manifestDependency struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	SourceURL      string   `json:"sourceUrl"`
	SourceURLs     []string `json:"sourceUrls"`
	ChecksumSHA256 string   `json:"checksumSha256"`
	SizeBytes      int64    `json:"sizeBytes"`
}

func (d manifestDependency) getFirstURL() string {
	if len(d.SourceURLs) > 0 {
		return d.SourceURLs[0]
	}
	return d.SourceURL
}

type manifestOption struct {
	Value        string   `json:"value"`
	Dependencies []string `json:"dependencies"`
}

func (o *manifestOption) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &o.Value); err == nil {
		return nil
	}
	type plain manifestOption
	return json.Unmarshal(data, (*plain)(o))
}

type modelManifest struct {
	Dependencies []manifestDependency `json:"dependencies"`
	Settings     []struct {
		Key     string           `json:"key"`
		Options []manifestOption `json:"options"`
	} `json:"settings"`
}

type mirrorInventory struct {
	Assets []struct {
		ID     string `json:"id"`
		SHA256 string `json:"sha256"`
	} `json:"assets"`
}

func readRecordedChecksums(t *testing.T) map[string]string {
	t.Helper()
	data, err := os.ReadFile("../../tools/plugin-mirror/inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	var inventory mirrorInventory
	if err := json.Unmarshal(data, &inventory); err != nil {
		t.Fatal(err)
	}
	checksums := map[string]string{}
	for _, asset := range inventory.Assets {
		checksums[asset.ID] = asset.SHA256
	}
	return checksums
}

func readModelManifest(t *testing.T) modelManifest {
	t.Helper()
	var m modelManifest
	if err := json.Unmarshal(Manifest, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func checkDeclaredModels(t *testing.T, pinned map[string]string) {
	t.Helper()
	checksums := readRecordedChecksums(t)
	declared := map[string]bool{}
	for _, dependency := range readModelManifest(t).Dependencies {
		if dependency.ID == "onnxruntime" {
			if dependency.Kind != "runtime" {
				t.Errorf("onnxruntime has kind %q, want runtime", dependency.Kind)
			}
			continue
		}
		address, ok := pinned[dependency.ID]
		if !ok {
			t.Errorf("the manifest declares %s, which no model of the plugin uses", dependency.ID)
			continue
		}
		declared[dependency.ID] = true
		sum := checksums[dependency.ID]
		if sum == "" {
			t.Errorf("the mirror inventory records no checksum of %s", dependency.ID)
		}
		if dependency.Kind != "model" || dependency.getFirstURL() != address || dependency.ChecksumSHA256 != sum || dependency.SizeBytes <= 0 {
			t.Errorf("%s = %+v, want a model from %s with sha256 %s and a size", dependency.ID, dependency, address, sum)
		}
	}
	for id := range pinned {
		if !declared[id] {
			t.Errorf("the manifest does not declare the model %s", id)
		}
	}
}

func checkOptionModels(t *testing.T, key string, want map[string][]string) {
	t.Helper()
	for _, setting := range readModelManifest(t).Settings {
		if setting.Key != key {
			continue
		}
		if len(setting.Options) != len(want) {
			t.Errorf("the setting %s offers %d options, want %d", key, len(setting.Options), len(want))
		}
		for _, option := range setting.Options {
			got, expected := slices.Sorted(slices.Values(option.Dependencies)), slices.Sorted(slices.Values(want[option.Value]))
			if !slices.Equal(got, expected) {
				t.Errorf("option %s names %v, want %v", option.Value, got, expected)
			}
		}
		return
	}
	t.Fatalf("the manifest has no setting %s", key)
}

const (
	parakeetV3 = "https://huggingface.co/csukuangfj/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8/resolve/2bda32ec70b097a55adaa07d9a7173915b43cc78/"
	parakeetV2 = "https://huggingface.co/csukuangfj/sherpa-onnx-nemo-parakeet-tdt-0.6b-v2-int8/resolve/1ab9323565ddb038682214b292f588070a538ce2/"
)

func getPinnedModels() map[string]string {
	return map[string]string{
		"parakeet-v3-encoder": parakeetV3 + "encoder.int8.onnx",
		"parakeet-v3-decoder": parakeetV3 + "decoder.int8.onnx",
		"parakeet-v3-joiner":  parakeetV3 + "joiner.int8.onnx",
		"parakeet-v3-tokens":  parakeetV3 + "tokens.txt",
		"parakeet-v2-encoder": parakeetV2 + "encoder.int8.onnx",
		"parakeet-v2-decoder": parakeetV2 + "decoder.int8.onnx",
		"parakeet-v2-joiner":  parakeetV2 + "joiner.int8.onnx",
		"parakeet-v2-tokens":  parakeetV2 + "tokens.txt",
		"silero-vad":          "https://huggingface.co/onnx-community/silero-vad/resolve/e71cae966052b992a7eca6b17738916ce0eca4ec/onnx/model.onnx",
	}
}

func TestManifestDeclaresEveryModelWithItsSourceChecksumAndSize(t *testing.T) {
	checkDeclaredModels(t, getPinnedModels())
}

func TestEachSpeechModelOptionNamesTheFilesOfItsModel(t *testing.T) {
	checkOptionModels(t, "model", map[string][]string{
		"parakeet-tdt-0.6b-v3-int8": {"parakeet-v3-encoder", "parakeet-v3-decoder", "parakeet-v3-joiner", "parakeet-v3-tokens"},
		"parakeet-tdt-0.6b-v2-int8": {"parakeet-v2-encoder", "parakeet-v2-decoder", "parakeet-v2-joiner", "parakeet-v2-tokens"},
	})
}
