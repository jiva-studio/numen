package manifest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

const handleManifestHead = `{"id":"org.example.copies","name":"A","version":"1.0.0",` +
	`"backend": {"entrypoint": "bin/copies"},"contributes":{"noteTypes":[{"type":"link"}],"handles":`

func TestReadManifestReadsTheHandlesOfAPlugin(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(handleManifestHead +
		`[{"purpose":"first-purpose","contentTypes":["video/*"]},{"purpose":"second-purpose","noteTypes":["link"]},{"purpose":"third-purpose"}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	first, ok := m.GetHandle("first-purpose")
	if !ok || !first.IsMatching("video/mp4", "") || first.IsMatching("audio/mpeg", "") || first.HasNoteTypes() {
		t.Errorf("the handle of the first purpose is %+v", first)
	}
	second, ok := m.GetHandle("second-purpose")
	if !ok || !second.IsMatching("text/markdown", "link") || second.IsMatching("text/markdown", domain.TypeNote) || !second.HasNoteTypes() {
		t.Errorf("the handle of the second purpose is %+v", second)
	}
	third, ok := m.GetHandle("third-purpose")
	if !ok || len(third.ContentTypes) != 0 || len(third.NoteTypes) != 0 {
		t.Errorf("the handle of the third purpose is %+v", third)
	}
	if _, ok := m.GetHandle("fourth-purpose"); ok {
		t.Error("a purpose the plugin does not declare has a handle")
	}
	if got := len(m.Contributes.Handles); got != 3 || !slices.ContainsFunc(m.Contributes.Handles, func(h manifest.Handle) bool { return h.Purpose == "third-purpose" }) {
		t.Errorf("the handles came out as %+v", m.Contributes.Handles)
	}
}

func TestReadManifest_RejectsInvalidHandle(t *testing.T) {
	for _, handles := range []string{
		`[{"contentTypes":["video/*"]}]`, `[{"purpose":"Upper"}]`, `[{"purpose":"a"},{"purpose":"a"}]`,
		`[{"purpose":"a","contentTypes":[".url"]}]`, `[{"purpose":"a","noteTypes":["deck"]}]`,
	} {
		_, err := manifestjson.Read(strings.NewReader(handleManifestHead + handles + `}}`))
		if !errors.Is(err, manifest.ErrInvalidHandle) {
			t.Errorf("handles %s read as %v; want ErrInvalidHandle", handles, err)
		}
	}
}

func TestAPluginThatDeclaresHandlesHasABackendToServeThem(t *testing.T) {
	_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0",
		"contributes":{"handles":[{"purpose":"sample-purpose"}]}}`))
	if !errors.Is(err, manifest.ErrNoBackend) {
		t.Errorf("error = %v, want ErrNoBackend", err)
	}
}
