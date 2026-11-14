package link_test

import (
	"errors"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

func (w *world) listCopies(t *testing.T) []string {
	t.Helper()
	files, err := w.store.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, one := range files {
		if ext := path.Ext(one.Name); ext == ".mp4" || ext == ".m4a" || ext == ".part" {
			out = append(out, one.Name)
		}
	}
	return out
}

func (w *world) copyShared(t *testing.T) (link.Link, link.Link) {
	t.Helper()
	first, second := w.createIn(t, "links", videoAddress), w.createIn(t, "other", videoAddress)
	for _, one := range []link.Link{first, second} {
		if _, err := w.links.Copy(t.Context(), one, fetch.MediaVideo); err != nil {
			t.Fatal(err)
		}
	}
	return first, second
}

func TestTwoNotesOfOneAddressShareOneCopy(t *testing.T) {
	w := newWorld(t)
	first, second := w.copyShared(t)

	if len(w.sites.media) != 1 {
		t.Errorf("the site was asked %v, want one download", w.sites.media)
	}
	if got, want := w.readProperty(t, second.Path, "media"), w.readProperty(t, first.Path, "media"); got != want || got == "" {
		t.Errorf("the second note holds media %q, the first %q", got, want)
	}
	if copies := w.listCopies(t); len(copies) != 1 {
		t.Errorf("the store holds copies %v, want one", copies)
	}
}

func TestDeleteCopyKeepsTheFileAnotherNoteRefersTo(t *testing.T) {
	w := newWorld(t)
	first, second := w.copyShared(t)
	media := w.readProperty(t, second.Path, "media")

	if err := w.links.DeleteCopy(t.Context(), first); err != nil {
		t.Fatal(err)
	}

	if got := w.readProperty(t, first.Path, "media"); got != "" {
		t.Errorf("the note whose media was deleted holds %q", got)
	}
	if got := w.readProperty(t, second.Path, "media"); got != media {
		t.Errorf("the other note holds media %q, want %q", got, media)
	}
	if copies := w.listCopies(t); len(copies) != 1 {
		t.Fatalf("the store holds copies %v, want the shared one", copies)
	}

	if err := w.links.DeleteCopy(t.Context(), second); err != nil {
		t.Fatal(err)
	}
	if copies := w.listCopies(t); len(copies) != 0 {
		t.Errorf("no note refers to the copies %v, and they are kept", copies)
	}
}

func TestASwitchToAudioKeepsTheVideoAnotherNoteRefersTo(t *testing.T) {
	w := newWorld(t)
	first, second := w.copyShared(t)
	video := w.readProperty(t, second.Path, "media")

	if _, err := w.links.Copy(t.Context(), first, fetch.MediaAudio); err != nil {
		t.Fatal(err)
	}

	audio := w.readProperty(t, first.Path, "media")
	if audio == video || audio == "" {
		t.Errorf("the switched note holds media %q, the video is %q", audio, video)
	}
	if got := w.readProperty(t, second.Path, "media"); got != video {
		t.Errorf("the other note holds media %q, want the video %q", got, video)
	}
	if copies := w.listCopies(t); len(copies) != 2 {
		t.Fatalf("the store holds copies %v, want the video and the sound", copies)
	}

	if _, err := w.links.Copy(t.Context(), second, fetch.MediaAudio); err != nil {
		t.Fatal(err)
	}
	if got := w.readProperty(t, second.Path, "media"); got != audio {
		t.Errorf("the second note holds media %q, want the shared sound %q", got, audio)
	}
	if copies := w.listCopies(t); len(copies) != 1 || len(w.sites.media) != 2 {
		t.Errorf("the store holds copies %v after %v downloads, want the sound alone after two", copies, w.sites.media)
	}
}

func TestDeleteCopyWaitsForTheRunningDownload(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)
	w.sites.started, w.sites.gate = make(chan struct{}), make(chan struct{})
	release := sync.OnceFunc(func() { close(w.sites.gate) })
	t.Cleanup(release)
	if _, isStarted, err := w.links.StartMediaCopy(t.Context(), made, fetch.MediaVideo); err != nil || !isStarted {
		t.Fatalf("StartMediaCopy = %v, %v", isStarted, err)
	}
	<-w.sites.started

	deleted := make(chan error, 1)
	go func() { deleted <- w.links.DeleteCopy(t.Context(), made) }()
	select {
	case err := <-deleted:
		t.Fatalf("DeleteCopy returned %v while the download ran", err)
	case <-time.After(50 * time.Millisecond):
	}
	release()

	if err := <-deleted; err != nil {
		t.Fatal(err)
	}
	w.links.Wait()
	if got := w.readProperty(t, made.Path, "media"); got != "" {
		t.Errorf("the note holds media %q after the delete", got)
	}
	if copies := w.listCopies(t); len(copies) != 0 {
		t.Errorf("the store holds %v after the delete", copies)
	}
	if _, err := w.store.Read(t.Context(), made.Key+".part"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("the partial file is left: %v", err)
	}
}
