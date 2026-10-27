package editor

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/port/playbackport"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// newMediaClient is the media of a window as a client reaches them, over a window whose
// loopback socket is open.
func newMediaClient(t *testing.T) numenv1connect.MediaServiceClient {
	t.Helper()
	api, _ := openRecordingWindow(t, nil)
	back, err := Listen(api.server, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { back.Close() })
	api.openPlaying(back)
	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	return numenv1connect.NewMediaServiceClient(server.Client(), server.URL)
}

func TestGetMediaSourceReturnsLoopbackAddress(t *testing.T) {
	client := newMediaClient(t)

	answer, err := client.GetMediaSource(t.Context(), connect.NewRequest(&v1.GetMediaSourceRequest{Path: talk}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetMediaType() != "audio/mpeg" {
		t.Errorf("a recording is played as %q, want audio/mpeg", answer.Msg.GetMediaType())
	}
	played, err := http.Get(answer.Msg.GetSrc())
	if err != nil {
		t.Fatal(err)
	}
	defer played.Body.Close()
	body, err := io.ReadAll(played.Body)
	if err != nil || string(body) != sound {
		t.Errorf("the address served %q (%v), want the recording's bytes", body, err)
	}
}

func TestGetMediaSourceRejectsMissingFile(t *testing.T) {
	client := newMediaClient(t)

	_, err := client.GetMediaSource(t.Context(), connect.NewRequest(&v1.GetMediaSourceRequest{Path: "talks/none.mp3"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("a missing file was answered %v, want not found", err)
	}
}

// keptCopies keeps a copy of each link note, under the hash of the note's content.
type keptCopies map[string][]byte

func (keptCopies) IsCopyingNotes() bool { return true }

func (keptCopies) IsCopied(_ string, noteType domain.NoteType) bool { return noteType == "link" }

func (k keptCopies) OpenCopy(_ context.Context, _ domain.Vault, _ string, _ domain.NoteType, hash string) (playbackport.Copy, error) {
	content, ok := k[hash]
	if !ok {
		return playbackport.Copy{}, playbackport.ErrNoCopy
	}
	return playbackport.Copy{Body: nopSeekCloser{bytes.NewReader(content)}, Size: int64(len(content)), MediaType: "video/mp4"}, nil
}

type nopSeekCloser struct{ io.ReadSeeker }

func (nopSeekCloser) Close() error { return nil }

const (
	clip      = "videos/clip.md"
	clipLink  = "---\ntype: link\nurl: https://videos.example/clip\n---\n"
	clipBytes = "the bytes of a copy of a video, long enough to ask for a piece of"
	plain     = "notes/plain.md"
	plainNote = "# A plain note\n"
)

// newCopyClient is the media of a window whose vault holds a note a plugin keeps a copy of.
func newCopyClient(t *testing.T, copies keptCopies) numenv1connect.MediaServiceClient {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{clip: clipLink, talk: sound, plain: plainNote})
	api := newBareServices()
	readers := vaultreader.Readers{}
	api.readFiles(readers, file.NewOpenCopy(readers, copies))
	api.state.setVault(vault)
	back, err := Listen(api.server, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { back.Close() })
	api.openPlaying(back)
	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	return numenv1connect.NewMediaServiceClient(server.Client(), server.URL)
}

func TestGetMediaSourcePlaysTheCopyAPluginKeepsOfTheFile(t *testing.T) {
	copies := keptCopies{domain.HashContent([]byte(clipLink)): []byte(clipBytes)}
	client := newCopyClient(t, copies)

	answer, err := client.GetMediaSource(t.Context(), connect.NewRequest(&v1.GetMediaSourceRequest{Path: clip}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetMediaType() != "video/mp4" {
		t.Errorf("the copy is played as %q, want video/mp4", answer.Msg.GetMediaType())
	}
	asked, err := http.NewRequest(http.MethodGet, answer.Msg.GetSrc(), nil)
	if err != nil {
		t.Fatal(err)
	}
	asked.Header.Set("Range", "bytes=4-12")
	played, err := http.DefaultClient.Do(asked)
	if err != nil {
		t.Fatal(err)
	}
	defer played.Body.Close()
	body, err := io.ReadAll(played.Body)
	if err != nil || played.StatusCode != http.StatusPartialContent || string(body) != clipBytes[4:13] {
		t.Errorf("the address answered %d with %q (%v), want a piece of the copy", played.StatusCode, body, err)
	}
}

func TestAnAddressOfACopyMadeAgainIsRefused(t *testing.T) {
	hash := domain.HashContent([]byte(clipLink))
	copies := keptCopies{hash: []byte(clipBytes)}
	client := newCopyClient(t, copies)
	answer, err := client.GetMediaSource(t.Context(), connect.NewRequest(&v1.GetMediaSourceRequest{Path: clip}))
	if err != nil {
		t.Fatal(err)
	}

	copies[hash] = []byte("a shorter copy")
	played, err := http.Get(answer.Msg.GetSrc())
	if err != nil {
		t.Fatal(err)
	}
	played.Body.Close()
	if played.StatusCode != http.StatusNotFound {
		t.Errorf("an address of the copy before was answered %d", played.StatusCode)
	}
}

func TestAFileWithNoCopyIsPlayedAsItself(t *testing.T) {
	client := newCopyClient(t, keptCopies{})

	answer, err := client.GetMediaSource(t.Context(), connect.NewRequest(&v1.GetMediaSourceRequest{Path: talk}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetMediaType() != "audio/mpeg" {
		t.Errorf("a recording is played as %q, want audio/mpeg", answer.Msg.GetMediaType())
	}
}

func TestGetMediaSourcePlaysAPlainNoteAsItself(t *testing.T) {
	client := newCopyClient(t, keptCopies{domain.HashContent([]byte(plainNote)): []byte(clipBytes)})

	answer, err := client.GetMediaSource(t.Context(), connect.NewRequest(&v1.GetMediaSourceRequest{Path: plain}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetMediaType() == "video/mp4" {
		t.Error("a plain note is played from a copy, and no plugin keeps copies of plain notes")
	}
}
