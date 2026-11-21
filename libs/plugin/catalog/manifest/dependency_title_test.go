package manifest_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func TestModelFileIsNamedByItsAddress(t *testing.T) {
	sum := strings.Repeat("a", 64)
	cases := map[string]string{
		"https://huggingface.co/Org/Layout_onnx/resolve/abc/inference.onnx": "Layout_onnx model",
		"https://example.org/models/rec/latin_rec.onnx":                     "latin_rec model",
		"https://example.org/inference.onnx":                                "example.org model",
	}
	for address, want := range cases {
		req, err := manifest.NewModelFile(address, sum)
		if err != nil {
			t.Fatal(err)
		}
		if got := req.GetTitle(); got != want {
			t.Errorf("title of %s = %q, want %q", address, got, want)
		}
	}
	if got := (manifest.DependencyRequirement{ID: "whisper"}).GetTitle(); got != "whisper" {
		t.Errorf("title without metadata = %q", got)
	}
}
