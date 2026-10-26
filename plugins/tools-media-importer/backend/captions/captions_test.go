package captions_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/captions"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
)

func TestCaptionsRoundTripThroughWebVTT(t *testing.T) {
	encoded := captions.Marshal([]download.Cue{
		{Text: "first caption", From: 1500, To: 4200},
		{Text: "  ", From: 4200, To: 5000},
		{Text: "second caption", From: 3723456, To: 3725000},
	})

	want := []download.Cue{
		{Text: "first caption", From: 1500, To: 4200},
		{Text: "second caption", From: 3723456, To: 3725000},
	}
	if got := captions.Parse(encoded); !slices.Equal(got, want) {
		t.Errorf("the cues are %+v, want %+v", got, want)
	}
}

func TestANoteANamedCueAndTimingsOtherToolsWriteAreRead(t *testing.T) {
	cues := captions.Parse([]byte("WEBVTT\r\n\r\nNOTE encoded elsewhere\r\n\r\ncue-1\r\n01:02.500 --> 00:01:03.000 line:0\r\nsaid\r\n"))

	if len(cues) != 1 || cues[0] != (download.Cue{Text: "said", From: 62500, To: 63000}) {
		t.Errorf("read %+v", cues)
	}
}
