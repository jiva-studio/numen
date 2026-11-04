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
	nt, ok := m.GetNoteType("link")
	if !ok || nt.Type != "link" {
		t.Errorf("GetNoteType(link) = %+v, %v", nt, ok)
	}
	if _, ok := m.GetNoteType("missing"); ok {
		t.Error("GetNoteType(missing) should be false")
	}
}

func TestNoteTypeValidationAndProfile(t *testing.T) {
	chunks := false
	headings := []int{1, 2}
	nt := manifest.NoteType{Type: "link", Chunks: &chunks, Headings: &headings, IDHeadingLevels: []int{3}, IDPattern: "[a-z]+"}
	profile := nt.GetProfile()
	if profile.HasChunks || !slices.Equal(profile.HeadingLevels, []int{1, 2}) || !slices.Equal(profile.IDHeadingLevels, []int{3}) || profile.IDPattern == nil {
		t.Errorf("profile = %+v", profile)
	}
	var nilHeadings []int
	emptyProfile := (manifest.NoteType{Headings: &nilHeadings}).GetProfile()
	if emptyProfile.HeadingLevels == nil || len(emptyProfile.HeadingLevels) != 0 {
		t.Errorf("empty headings profile = %+v", emptyProfile)
	}

	for name, noteTypes := range map[string]string{
		"invalid type":       `[{"type":"bad/type"}]`,
		"duplicate type":     `[{"type":"link"},{"type":"link"}]`,
		"invalid pattern":    `[{"type":"link","idPattern":"["}]`,
		"heading level low":  `[{"type":"link","headings":[0]}]`,
		"heading level high": `[{"type":"link","headings":[7]}]`,
		"id level low":       `[{"type":"link","idHeadingLevels":[0]}]`,
		"id level high":      `[{"type":"link","idHeadingLevels":[7]}]`,
	} {
		_, err := manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","contributes":{"noteTypes":` + noteTypes + `}}`))
		if !errors.Is(err, manifest.ErrInvalidNoteType) {
			t.Errorf("%s: got %v, want ErrInvalidNoteType", name, err)
		}
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
