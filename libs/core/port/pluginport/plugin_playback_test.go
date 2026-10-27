package pluginport_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const playbackManifestHead = `{"id":"org.example.copies","name":"A","version":"1.0.0",` +
	`"backend":{"type":"process","entrypoint":"bin/copies"}`

func TestReadManifestReadsPlaybackContentTypes(t *testing.T) {
	m, err := pluginport.ReadManifest(strings.NewReader(playbackManifestHead + `,
		"playback": {"contentTypes": ["text/markdown"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !m.Playback.IsCopying("text/markdown") || m.Playback.IsCopying("audio/mpeg") {
		t.Error("IsCopying answers for the wrong types")
	}
}

func TestReadManifestRejectsPlaybackByExtension(t *testing.T) {
	for _, playback := range []string{`{"contentTypes": []}`, `{"contentTypes": [".url"]}`, `{"files": [".url"]}`} {
		_, err := pluginport.ReadManifest(strings.NewReader(playbackManifestHead + `, "playback": ` + playback + `}`))
		if !errors.Is(err, pluginport.ErrInvalidPlayback) {
			t.Errorf("playback %s read as %v; want ErrInvalidPlayback", playback, err)
		}
	}
}
