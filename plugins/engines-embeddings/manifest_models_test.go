package embeddings

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
	e5Small     = "https://huggingface.co/Xenova/multilingual-e5-small/resolve/761b726dd34fb83930e26aab4e9ac3899aa1fa78/"
	harrier270m = "https://huggingface.co/onnx-community/harrier-oss-v1-270m-ONNX/resolve/d59c919d0159aea2c19ed7d04288fcdd048d0f9c/"
	bgeM3       = "https://huggingface.co/Xenova/bge-m3/resolve/4de13258303883538bd53b696b452bf8099f0858/"
)

func getPinnedModels() map[string]string {
	return map[string]string{
		"e5-small-graph":          e5Small + "onnx/model_quantized.onnx",
		"e5-small-tokenizer":      e5Small + "tokenizer.json",
		"harrier-270m-graph":      harrier270m + "onnx/model_quantized.onnx",
		"harrier-270m-graph-data": harrier270m + "onnx/model_quantized.onnx_data",
		"harrier-270m-tokenizer":  harrier270m + "tokenizer.json",
		"bge-m3-graph":            bgeM3 + "onnx/model_int8.onnx",
		"bge-m3-tokenizer":        bgeM3 + "tokenizer.json",
	}
}

func TestManifestDeclaresEveryModelWithItsSourceChecksumAndSize(t *testing.T) {
	checkDeclaredModels(t, getPinnedModels())
}

func TestEachCuratedModelOptionNamesTheFilesOfItsModel(t *testing.T) {
	checkOptionModels(t, "model", map[string][]string{
		"Xenova/multilingual-e5-small":            {"e5-small-graph", "e5-small-tokenizer"},
		"onnx-community/harrier-oss-v1-270m-ONNX": {"harrier-270m-graph", "harrier-270m-graph-data", "harrier-270m-tokenizer"},
		"Xenova/bge-m3":                           {"bge-m3-graph", "bge-m3-tokenizer"},
	})
}
