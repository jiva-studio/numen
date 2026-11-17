package link_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
)

func TestCopyOfTheAudioDownloadsTheSoundAlone(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)

	if _, err := w.links.Copy(t.Context(), made, fetch.MediaAudio); err != nil {
		t.Fatal(err)
	}

	if want := []fetch.Media{fetch.MediaAudio}; len(w.sites.media) != 1 || w.sites.media[0] != want[0] {
		t.Errorf("the site was asked for %v, want %v", w.sites.media, want)
	}
	copied, err := w.store.Read(t.Context(), made.Key+".audio.m4a")
	if err != nil || !bytes.Equal(copied, w.sites.clip) {
		t.Errorf("the folder keeps %d bytes of sound (%v)", len(copied), err)
	}
	if want, got := "plugin://"+pluginID+"/"+made.Key+".audio.m4a", w.readProperty(t, made.Path, "media"); got != want {
		t.Errorf("the note holds media %q, want %q", got, want)
	}
}

func TestCopyOfTheVideoReplacesTheSoundDownloadedBefore(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)
	if _, err := w.links.Copy(t.Context(), made, fetch.MediaAudio); err != nil {
		t.Fatal(err)
	}

	if _, err := w.links.Copy(t.Context(), made, fetch.MediaVideo); err != nil {
		t.Fatal(err)
	}

	if _, err := w.store.Read(t.Context(), made.Key+".audio.m4a"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("the sound is still kept: %v", err)
	}
	if want, got := "plugin://"+pluginID+"/"+made.Key+".video.mp4", w.readProperty(t, made.Path, "media"); got != want {
		t.Errorf("the note holds media %q, want %q", got, want)
	}
	if _, err := w.links.Copy(t.Context(), made, fetch.MediaVideo); err != nil || len(w.sites.media) != 2 {
		t.Errorf("a second copy of the video asked the site %v times (%v)", w.sites.media, err)
	}
}
