package link_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
)

// A copy of the other media that fails leaves the note with media it can play.
func TestCopyOfTheVideoThatFailsKeepsTheSoundTheNotePlays(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)
	if _, err := w.links.Copy(t.Context(), made, download.MediaAudio); err != nil {
		t.Fatal(err)
	}
	media := w.readProperty(t, made.Path, "media")

	w.sites.failAt = 100
	if _, err := w.links.Copy(t.Context(), made, download.MediaVideo); err == nil {
		t.Fatal("a copy the site hung up on was kept")
	}

	if got := w.readProperty(t, made.Path, "media"); got == media {
		if _, err := w.store.Read(t.Context(), made.Key+".audio.m4a"); err != nil {
			t.Errorf("the note holds media %q, and its file is gone: %v", got, err)
		}
	}
}
