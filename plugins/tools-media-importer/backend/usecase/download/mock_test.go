package download

import (
	"bytes"
	"encoding/binary"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/ytdlp"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

func TestTheMockCopiesAClipOfAVideoSiteAndPublishesItsCaptions(t *testing.T) {
	mock := NewMock(fakeWeb{page: ""})
	const clip = "https://www.youtube.com/watch?v=tidalmill01"
	if !mock.CanCopy(clip) || mock.CanCopy("http://tidal-mills.numen.test/history/rance") {
		t.Fatal("the mock copies what is not a video, or does not copy a video")
	}
	text, err := mock.GetText(t.Context(), clip, fetch.PreferredCaptions{})
	wantCues := []ytdlp.Cue{
		{Text: "The tide comes in twice a day.", From: 0, To: 1000},
		{Text: "The mill pond fills behind the gate.", From: 1000, To: 2000},
		{Text: "On the ebb the wheel turns.", From: 2000, To: 3000},
	}
	if err != nil || text.Producer != "captions" || text.Length != 3000 || !slices.Equal(text.Cues, wantCues) {
		t.Fatalf("the captions are %+v (%v)", text, err)
	}
	if _, err := mock.Download(t.Context(), clip, fetch.MediaVideo, failingWriter{}); err == nil {
		t.Fatal("expected error from failing writer")
	}
	var into bytes.Buffer
	copied, err := mock.Download(t.Context(), clip, fetch.MediaAudio, &into)
	if err != nil {
		t.Fatal(err)
	}
	raw := into.Bytes()
	const wantSamples = 8000 * 3
	if len(raw) != 44+wantSamples {
		t.Fatalf("wav length = %d, want %d", len(raw), 44+wantSamples)
	}
	if copied.Extension != ".wav" || string(raw[:4]) != "RIFF" || string(raw[8:16]) != "WAVEfmt " || string(raw[36:40]) != "data" {
		t.Errorf("the copy is %+v, header %q", copied, raw[:44])
	}
	if got := binary.LittleEndian.Uint32(raw[4:8]); got != 36+wantSamples {
		t.Errorf("chunk size = %d, want %d", got, 36+wantSamples)
	}
	if got := binary.LittleEndian.Uint32(raw[16:20]); got != 16 {
		t.Errorf("fmt size = %d, want 16", got)
	}
	if got := binary.LittleEndian.Uint16(raw[20:22]); got != 1 {
		t.Errorf("audio format = %d, want 1", got)
	}
	if got := binary.LittleEndian.Uint16(raw[22:24]); got != 1 {
		t.Errorf("channels = %d, want 1", got)
	}
	if got := binary.LittleEndian.Uint32(raw[24:28]); got != 8000 {
		t.Errorf("sample rate = %d, want 8000", got)
	}
	if got := binary.LittleEndian.Uint32(raw[28:32]); got != 8000 {
		t.Errorf("byte rate = %d, want 8000", got)
	}
	if got := binary.LittleEndian.Uint16(raw[32:34]); got != 1 {
		t.Errorf("block align = %d, want 1", got)
	}
	if got := binary.LittleEndian.Uint16(raw[34:36]); got != 8 {
		t.Errorf("bits per sample = %d, want 8", got)
	}
	if got := binary.LittleEndian.Uint32(raw[40:44]); got != wantSamples {
		t.Errorf("data size = %d, want %d", got, wantSamples)
	}
	for i := 44; i < len(raw); i++ {
		if raw[i] != 128 {
			t.Fatalf("sample at %d = %d, want 128", i, raw[i])
		}
	}
	meta, err := mock.GetMetadata(t.Context(), clip)
	if err != nil || meta.Bytes != int64(len(raw)) || meta.Length != 3000 {
		t.Errorf("metadata = %+v, len(raw) = %d (%v)", meta, len(raw), err)
	}
}
