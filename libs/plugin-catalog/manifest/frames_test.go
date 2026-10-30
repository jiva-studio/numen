package manifest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

const framesManifestHead = `{"id":"org.example.frames","name":"A","version":"1.0.0",` +
	`"backend":{"type":"process","entrypoint":"bin/frames"}`

func TestReadManifestReadsTheOriginsThePluginFrames(t *testing.T) {
	m, err := manifestjson.Read(strings.NewReader(framesManifestHead + `,
		"frames": ["https://player.example", "https://video.example:8443"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Frames, []string{"https://player.example", "https://video.example:8443"}) {
		t.Errorf("the frames came out as %v", m.Frames)
	}
}

func TestReadManifestRejectsAFrameThatIsNoHTTPSOrigin(t *testing.T) {
	for _, frame := range []string{
		"http://player.example", "https://player.example/embed", "https://player.example?a=1",
		"player.example", "*", "https://", "https://user@player.example",
	} {
		_, err := manifestjson.Read(strings.NewReader(framesManifestHead + `, "frames": ["` + frame + `"]}`))
		if !errors.Is(err, manifest.ErrInvalidFrame) {
			t.Errorf("frame %q read as %v; want ErrInvalidFrame", frame, err)
		}
	}
}
