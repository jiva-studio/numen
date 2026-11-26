package ocr

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

const rapidOCR = "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/4e9666f0fbf5fac850d30e530bf0a2e0d886aa78/onnx/"

func getPinnedModels() map[string]string {
	return map[string]string{
		"pp-doclayout":            "https://huggingface.co/PaddlePaddle/PP-DocLayoutV3_onnx/resolve/46bbdf188bb0a772c08aed74882ce7e51a8f1ea6/inference.onnx",
		"pp-ocr-det":              "https://huggingface.co/PaddlePaddle/PP-OCRv5_mobile_det_onnx/resolve/e6f4fa85f00e168c862bc462aebca69eef9b3d3d/inference.onnx",
		"rec-pp-ocrv6-tiny":       rapidOCR + "PP-OCRv6/rec/PP-OCRv6_rec_tiny.onnx",
		"rec-pp-ocrv6-small":      rapidOCR + "PP-OCRv6/rec/PP-OCRv6_rec_small.onnx",
		"rec-pp-ocrv5-latin":      rapidOCR + "PP-OCRv5/rec/latin_PP-OCRv5_rec_mobile.onnx",
		"rec-pp-ocrv5-eslav":      rapidOCR + "PP-OCRv5/rec/eslav_PP-OCRv5_rec_mobile.onnx",
		"rec-pp-ocrv5-cyrillic":   rapidOCR + "PP-OCRv5/rec/cyrillic_PP-OCRv5_rec_mobile.onnx",
		"rec-pp-ocrv5-devanagari": rapidOCR + "PP-OCRv5/rec/devanagari_PP-OCRv5_rec_mobile.onnx",
		"rec-pp-ocrv5-greek":      rapidOCR + "PP-OCRv5/rec/el_PP-OCRv5_rec_mobile.onnx",
	}
}

func TestManifestDeclaresEveryModelWithItsSourceChecksumAndSize(t *testing.T) {
	checkDeclaredModels(t, getPinnedModels())
}

func TestEachRecognizerOptionNamesItsRecognitionModel(t *testing.T) {
	checkOptionModels(t, "model", map[string][]string{
		"pp-ocrv6-tiny":       {"rec-pp-ocrv6-tiny"},
		"pp-ocrv6-small":      {"rec-pp-ocrv6-small"},
		"pp-ocrv5-latin":      {"rec-pp-ocrv5-latin"},
		"pp-ocrv5-eslav":      {"rec-pp-ocrv5-eslav"},
		"pp-ocrv5-cyrillic":   {"rec-pp-ocrv5-cyrillic"},
		"pp-ocrv5-devanagari": {"rec-pp-ocrv5-devanagari"},
		"pp-ocrv5-greek":      {"rec-pp-ocrv5-greek"},
		"custom":              nil,
	})
}
