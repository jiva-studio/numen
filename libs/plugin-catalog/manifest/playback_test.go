package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

const playbackManifestHead = `{"id":"org.example.copies","name":"A","version":"1.0.0",` +
	`"backend":{"type":"process","entrypoint":"bin/copies"},"noteTypes":[{"type":"link"}]`

func TestReadManifestReadsPlaybackContentTypes(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(playbackManifestHead + `,
		"playback": {"contentTypes": ["video/*"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !m.Playback.IsCopying("video/mp4", "") || m.Playback.IsCopying("audio/mpeg", "") {
		t.Error("IsCopying answers for the wrong types")
	}
}

func TestReadManifestReadsPlaybackNoteTypes(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(playbackManifestHead + `,
		"playback": {"noteTypes": ["link"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !m.Playback.IsCopying("text/markdown", "link") || m.Playback.IsCopying("text/markdown", domain.TypeNote) {
		t.Error("IsCopying answers for the wrong notes")
	}
}

func TestReadManifest_RejectsInvalidPlayback(t *testing.T) {
	for _, playback := range []string{
		`{"contentTypes": []}`, `{"contentTypes": [".url"]}`, `{"files": [".url"]}`, `{"noteTypes": ["deck"]}`,
	} {
		_, err := manifestjson.Read(strings.NewReader(playbackManifestHead + `, "playback": ` + playback + `}`))
		if !errors.Is(err, manifest.ErrInvalidPlayback) {
			t.Errorf("playback %s read as %v; want ErrInvalidPlayback", playback, err)
		}
	}
}
