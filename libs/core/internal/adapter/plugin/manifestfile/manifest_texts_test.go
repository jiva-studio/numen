package manifestfile

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

func readTextsManifest(t *testing.T, texts string) (pluginport.PluginManifest, error) {
	t.Helper()
	return pluginport.ReadManifest(strings.NewReader(`{
		"id": "org.example.ocr", "name": "OCR", "version": "1.0.0", "backend": {"entrypoint": "bin/ocr"},
		"texts": ` + texts + `
	}`))
}

func TestAManifestDeclaresTheFilesItMakesTextOfOnItsOwn(t *testing.T) {
	m, err := readTextsManifest(t, `{"contentTypes": ["application/pdf", "image/vnd.djvu"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if m.Texts == nil || !slices.Equal(m.Texts.ContentTypes, []domain.ContentType{"application/pdf", "image/vnd.djvu"}) {
		t.Fatalf("read %+v", m.Texts)
	}
}

func TestAManifestDeclaresThatItMakesTextWhenACommandRuns(t *testing.T) {
	m, err := readTextsManifest(t, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if m.Texts == nil || len(m.Texts.ContentTypes) != 0 {
		t.Fatalf("read %+v", m.Texts)
	}
}

func TestATextsDeclarationThatNamesATypeWronglyIsRefused(t *testing.T) {
	for name, texts := range map[string]string{
		"an extension for a type": `{"contentTypes": [".pdf"]}`,
		"a type of no subtype":    `{"contentTypes": ["audio"]}`,
	} {
		if _, err := readTextsManifest(t, texts); !errors.Is(err, pluginport.ErrInvalidTexts) {
			t.Errorf("%s: got %v, want ErrInvalidTexts", name, err)
		}
	}
}
