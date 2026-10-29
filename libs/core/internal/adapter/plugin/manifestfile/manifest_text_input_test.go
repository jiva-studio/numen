package manifestfile

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
	m, err := readTextManifest(t, `{"contentTypes": ["application/pdf", "image/vnd.djvu"], "kinds": ["recognition"], "autoRun": true}`)
	if err != nil {
		t.Fatal(err)
	}
	in := m.TextInput
	if in == nil || !slices.Equal(in.ContentTypes, []domain.ContentType{"application/pdf", "image/vnd.djvu"}) || !in.IsAutoRun ||
		!slices.Equal(in.Kinds, []domain.TextKind{"recognition"}) {
		t.Fatalf("read %+v", in)
	}
}

func TestAManifestDeclaresItsFileMenuEntry(t *testing.T) {
	m, err := readTextManifest(t, `{"kinds": ["recognition"], "menuEntry": {"title": "Proofread", "icon": "spell-check"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.TextInput.MenuEntry; got == nil || *got != (pluginport.PluginMenuEntry{Title: "Proofread", Icon: "spell-check"}) {
		t.Fatalf("read the menu entry %+v", got)
	}
}

func TestATextInputThatNamesNothingOrNamesItWronglyIsRefused(t *testing.T) {
	for name, input := range map[string]string{
		"nothing":                         `{}`,
		"an extension for a type":         `{"contentTypes": [".pdf"]}`,
		"a type of no subtype":            `{"contentTypes": ["audio"]}`,
		"a kind with no name":             `{"kinds": [" "]}`,
		"a run on its own with no types":  `{"kinds": ["article"], "autoRun": true}`,
		"a menu entry with no title":      `{"kinds": ["article"], "menuEntry": {"title": " ", "icon": "spell-check"}}`,
		"a note type it does not declare": `{"noteTypes": ["link"]}`,
	} {
		if _, err := readTextManifest(t, input); !errors.Is(err, pluginport.ErrInvalidTextInput) {
			t.Errorf("%s: got %v, want ErrInvalidTextInput", name, err)
		}
	}
}

func TestATextInputNamesTheNoteTypesThePluginDeclares(t *testing.T) {
	m, err := pluginport.ReadManifest(strings.NewReader(`{
		"id": "org.example.links", "name": "Links", "version": "1.0.0", "backend": {"entrypoint": "bin/links"},
		"noteTypes": [{"type": "link"}],
		"textInput": {"noteTypes": ["link"]}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.TextInput.NoteTypes, []string{"link"}) {
		t.Errorf("read %+v", m.TextInput)
	}
}
