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
	m, err := readTextManifest(t, `{"contentTypes": ["application/pdf", "image/vnd.djvu"], "formats": ["positioned"], "autoRun": true}`)
	if err != nil {
		t.Fatal(err)
	}
	in := m.TextInput
	if in == nil || !slices.Equal(in.ContentTypes, []domain.ContentType{"application/pdf", "image/vnd.djvu"}) || !in.IsAutoRun ||
		!slices.Equal(in.Formats, []domain.TextFormat{domain.TextFormatPositioned}) {
		t.Fatalf("read %+v", in)
	}
}

func TestATextInputThatNamesNothingOrNamesItWronglyIsRefused(t *testing.T) {
	for name, input := range map[string]string{
		"nothing":                         `{}`,
		"an extension for a type":         `{"contentTypes": [".pdf"]}`,
		"a type of no subtype":            `{"contentTypes": ["audio"]}`,
		"a format the core does not read": `{"formats": ["html"]}`,
		"a run on its own with no types":  `{"formats": ["plain"], "autoRun": true}`,
	} {
		if _, err := readTextManifest(t, input); !errors.Is(err, pluginport.ErrInvalidTextInput) {
			t.Errorf("%s: got %v, want ErrInvalidTextInput", name, err)
		}
	}
}
