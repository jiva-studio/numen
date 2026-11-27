package reranking

import (
	"encoding/json"
	"os"
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

const (
	bgeRerankerV2M3 = "https://huggingface.co/onnx-community/bge-reranker-v2-m3-ONNX/resolve/6f5ff65298512715a1e669753bc754d2bc8f367b/"
	gteReranker     = "https://huggingface.co/onnx-community/gte-multilingual-reranker-base/resolve/ee64367e35a2db0da46bb6497e13a18f8bd585cb/"
)

func getPinnedModels() map[string]string {
	return map[string]string{
		"bge-reranker-v2-m3-graph":     bgeRerankerV2M3 + "onnx/model_quantized.onnx",
		"bge-reranker-v2-m3-tokenizer": bgeRerankerV2M3 + "tokenizer.json",
		"gte-reranker-graph":           gteReranker + "onnx/model_quantized.onnx",
		"gte-reranker-tokenizer":       gteReranker + "tokenizer.json",
	}
}

func TestManifestDeclaresEveryModelWithItsSourceChecksumAndSize(t *testing.T) {
	checkDeclaredModels(t, getPinnedModels())
}
