package link_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
)

func TestCopyOfTheAudioDownloadsTheSoundAlone(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)

	if _, err := w.links.Copy(t.Context(), made, download.MediaAudio); err != nil {
		t.Fatal(err)
	}

	if want := []download.Media{download.MediaAudio}; len(w.sites.media) != 1 || w.sites.media[0] != want[0] {
		t.Errorf("the site was asked for %v, want %v", w.sites.media, want)
	}
	copied, err := w.store.Read(t.Context(), made.Key+".m4a")
	if err != nil || !bytes.Equal(copied, w.sites.clip) {
		t.Errorf("the folder keeps %d bytes of sound (%v)", len(copied), err)
	}
	read, err := w.links.Read(t.Context(), made.Path)
	if want := "plugin://" + pluginID + "/" + made.Key + ".m4a"; err != nil || read.Media != want {
		t.Errorf("the note holds media %q (%v), want %q", read.Media, err, want)
	}
}

func TestCopyOfTheVideoReplacesTheSoundDownloadedBefore(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)
	if _, err := w.links.Copy(t.Context(), made, download.MediaAudio); err != nil {
		t.Fatal(err)
	}

	if _, err := w.links.Copy(t.Context(), made, download.MediaVideo); err != nil {
		t.Fatal(err)
	}

	if _, err := w.store.Read(t.Context(), made.Key+".m4a"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("the sound is still kept: %v", err)
	}
	read, err := w.links.Read(t.Context(), made.Path)
	if want := "plugin://" + pluginID + "/" + made.Key + ".mp4"; err != nil || read.Media != want {
		t.Errorf("the note holds media %q (%v), want %q", read.Media, err, want)
	}
	if _, err := w.links.Copy(t.Context(), made, download.MediaVideo); err != nil || len(w.sites.media) != 2 {
		t.Errorf("a second copy of the video asked the site %v times (%v)", w.sites.media, err)
	}
}
