package plugindir

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

func readTextManifest(t *testing.T, input string) (pluginport.PluginManifest, error) {
	t.Helper()
	return pluginport.ReadManifest(strings.NewReader(`{
		"id": "org.example.ocr", "name": "OCR", "version": "1.0.0", "backend": {"entrypoint": "bin/ocr"},
		"textInput": ` + input + `
	}`))
}

func TestAManifestDeclaresWhatItMakesTextFrom(t *testing.T) {
	m, err := readTextManifest(t, `{"files": [".pdf", ".djvu"], "formats": ["positioned"], "autoRun": true}`)
	if err != nil {
		t.Fatal(err)
	}
	in := m.TextInput
	if in == nil || !slices.Equal(in.Files, []string{".pdf", ".djvu"}) || !in.IsAutoRun ||
		!slices.Equal(in.Formats, []domain.TextFormat{domain.TextFormatPositioned}) {
		t.Fatalf("read %+v", in)
	}
}

func TestATextInputThatNamesNothingOrNamesItWronglyIsRefused(t *testing.T) {
	for name, input := range map[string]string{
		"nothing":                         `{}`,
		"a type with no dot":              `{"files": ["pdf"]}`,
		"a path":                          `{"files": [".a/b"]}`,
		"a format the core does not read": `{"formats": ["html"]}`,
		"a run on its own with no files":  `{"formats": ["plain"], "autoRun": true}`,
	} {
		if _, err := readTextManifest(t, input); !errors.Is(err, pluginport.ErrInvalidTextInput) {
			t.Errorf("%s: got %v, want ErrInvalidTextInput", name, err)
		}
	}
}
