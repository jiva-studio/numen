package hintsource_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/hintsource"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

func TestAFileIsMatchedToThePluginByExtensionOrName(t *testing.T) {
	source := hintsource.New([]hintsource.Hint{
		{ID: "org.example.reader", Name: "Reader", Description: "", Contributes: manifest.Contributes{TabKinds: nil, Feature: "", Category: ""}, Extensions: []string{"sample"}, FileNames: nil},
		{ID: "org.example.config", Name: "Config", Description: "", Contributes: manifest.Contributes{TabKinds: nil, Feature: "", Category: ""}, Extensions: nil, FileNames: []string{"sample.config"}},
	})
	cases := map[string]string{
		"book.sample":          "org.example.reader",
		"/vault/BOOK.SAMPLE":   "org.example.reader",
		`C:\vault\book.sample`: "org.example.reader",
		"sub/sample.config":    "org.example.config",
		"other.config":         "",
		"sample":               "",
	}
	for name, want := range cases {
		got, ok := source.FindByFile(name)

		if (want == "") == ok || got.ID != want {
			t.Errorf("FindByFile(%q) = %q, %v; want %q", name, got.ID, ok, want)
		}
	}
}

func TestHintsAreListedAsPluginsAvailableFromThisSource(t *testing.T) {
	source := hintsource.NewBuiltIn()

	entries, err := source.List(t.Context())
	got, getErr := source.Get(t.Context(), "md.numen.readers.pdf")
	_, missing := source.Get(t.Context(), "org.example.missing")

	if err != nil || len(entries) == 0 {
		t.Fatalf("List = %v, %v", entries, err)
	}
	for _, entry := range entries {
		if entry.State != catalog.StateAvailable || entry.Source != hintsource.Name || entry.ID == "" {
			t.Errorf("entry = %+v, want an available entry of the hint source", entry)
		}
	}
	if getErr != nil || got.Contributes.Feature == "" {
		t.Errorf("Get = %+v, %v", got, getErr)
	}
	if !errors.Is(missing, catalog.ErrNotFound) {
		t.Errorf("Get of a missing plugin: error = %v", missing)
	}
}

func TestEveryBuiltInHintIsValid(t *testing.T) {
	observedIDs := map[string]bool{}
	observedKinds := map[string]bool{}
	for _, hint := range hintsource.BuiltIn() {
		if hint.ID == "" || hint.Name == "" || hint.Contributes.Feature == "" || hint.Contributes.Category == "" || len(hint.Contributes.TabKinds) == 0 {
			t.Errorf("hint %q is incomplete: %+v", hint.ID, hint)
		}
		if observedIDs[hint.ID] {
			t.Errorf("hint %q is listed twice", hint.ID)
		}
		observedIDs[hint.ID] = true
		for _, kind := range hint.Contributes.TabKinds {
			if observedKinds[kind.Kind] {
				t.Errorf("tab kind %q is provided by two hints", kind.Kind)
			}
			observedKinds[kind.Kind] = true
		}
		for _, extension := range hint.Extensions {
			if extension == "" || slices.ContainsFunc([]rune(extension), func(r rune) bool { return r == '.' || (r >= 'A' && r <= 'Z') }) {
				t.Errorf("hint %q: extension %q is not lowercase without a dot", hint.ID, extension)
			}
		}
	}
}

func TestTheBuiltInHintsFindTheReadersByFileName(t *testing.T) {
	source := hintsource.NewBuiltIn()

	for name, id := range map[string]string{
		"book.epub": "md.numen.readers.epub",
		"paper.PDF": "md.numen.readers.pdf",
		"talk.mp3":  "md.numen.views.media.player",
		"note.md":   "md.numen.notes",
	} {
		if got, ok := source.FindByFile(name); !ok || got.ID != id {
			t.Errorf("FindByFile(%q) = %q, %v; want %q", name, got.ID, ok, id)
		}
	}
}
