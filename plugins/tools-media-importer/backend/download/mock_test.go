package download

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestTheMockCopiesAClipOfAVideoSiteAndPublishesItsCaptions(t *testing.T) {
	mock := NewMock()
	const clip = "https://www.youtube.com/watch?v=tidalmill01"
	if !mock.CanCopy(clip) || mock.CanCopy("http://tidal-mills.numen.test/history/rance") {
		t.Fatal("the mock copies what is not a video, or does not copy a video")
	}
	text, err := mock.GetText(t.Context(), clip, PreferredCaptions{})
	if err != nil || len(text.Cues) != 3 || text.Producer != "captions" {
		t.Fatalf("the captions are %+v (%v)", text, err)
	}
	var into bytes.Buffer
	copied, err := mock.Download(t.Context(), clip, &into)
	if err != nil {
		t.Fatal(err)
	}
	raw := into.Bytes()
	if copied.Extension != ".wav" || string(raw[:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		t.Errorf("the copy is %+v, beginning %q", copied, raw[:12])
	}
	if seconds := binary.LittleEndian.Uint32(raw[40:]) / binary.LittleEndian.Uint32(raw[28:]); seconds != mockSeconds {
		t.Errorf("the clip runs %d seconds", seconds)
	}
	meta, err := mock.GetMetadata(t.Context(), clip)
	if err != nil || meta.Bytes != int64(len(raw)) {
		t.Errorf("the clip is said to be %d bytes and is %d (%v)", meta.Bytes, len(raw), err)
	}
}
