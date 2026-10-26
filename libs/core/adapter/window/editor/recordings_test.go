package editor

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

// newRecordingClient is the recordings of a window as a client reaches them, over a window whose
// loopback socket is open.
func newRecordingClient(t *testing.T) numenv1connect.RecordingServiceClient {
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
	return numenv1connect.NewRecordingServiceClient(server.Client(), server.URL)
}

func TestGetRecordingReturnsLoopbackAddress(t *testing.T) {
	client := newRecordingClient(t)

	answer, err := client.GetRecording(t.Context(), connect.NewRequest(&v1.GetRecordingRequest{Path: talk}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetMediaType() != "audio/mpeg" {
		t.Errorf("a recording is played as %q, want audio/mpeg", answer.Msg.GetMediaType())
	}
	played, err := http.Get(answer.Msg.GetMediaUrl())
	if err != nil {
		t.Fatal(err)
	}
	defer played.Body.Close()
	body, err := io.ReadAll(played.Body)
	if err != nil || string(body) != sound {
		t.Errorf("the address served %q (%v), want the recording's bytes", body, err)
	}
}

func TestGetRecordingRejectsMissingFile(t *testing.T) {
	client := newRecordingClient(t)

	_, err := client.GetRecording(t.Context(), connect.NewRequest(&v1.GetRecordingRequest{Path: "talks/none.mp3"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("a missing file was answered %v, want not found", err)
	}
}

// keptCopies keeps a copy of each `.url` file, under the hash of the file's content.
type keptCopies map[string][]byte

func (keptCopies) IsCopied(path string) bool { return strings.HasSuffix(path, ".url") }

func (k keptCopies) OpenCopy(_ context.Context, _ domain.Vault, _, hash string) (playbackport.Copy, error) {
	content, ok := k[hash]
	if !ok {
		return playbackport.Copy{}, playbackport.ErrNoCopy
	}
	return playbackport.Copy{Body: nopSeekCloser{bytes.NewReader(content)}, Size: int64(len(content)), MediaType: "video/mp4"}, nil
}

type nopSeekCloser struct{ io.ReadSeeker }

func (nopSeekCloser) Close() error { return nil }

const (
	clip      = "videos/clip.url"
	clipLink  = "[InternetShortcut]\nURL=https://videos.example/clip\n"
	clipBytes = "the bytes of a copy of a video, long enough to ask for a piece of"
)

// newCopyClient is the recordings of a window whose vault holds a link a plugin keeps a copy of.
func newCopyClient(t *testing.T, copies keptCopies) numenv1connect.RecordingServiceClient {
	t.Helper()
	vault := testsupport.NewVault(t, map[string]string{clip: clipLink, talk: sound})
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
	return numenv1connect.NewRecordingServiceClient(server.Client(), server.URL)
}

func TestGetRecordingPlaysTheCopyAPluginKeepsOfTheFile(t *testing.T) {
	copies := keptCopies{domain.HashContent([]byte(clipLink)): []byte(clipBytes)}
	client := newCopyClient(t, copies)

	answer, err := client.GetRecording(t.Context(), connect.NewRequest(&v1.GetRecordingRequest{Path: clip}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetMediaType() != "video/mp4" {
		t.Errorf("the copy is played as %q, want video/mp4", answer.Msg.GetMediaType())
	}
	asked, err := http.NewRequest(http.MethodGet, answer.Msg.GetMediaUrl(), nil)
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
	answer, err := client.GetRecording(t.Context(), connect.NewRequest(&v1.GetRecordingRequest{Path: clip}))
	if err != nil {
		t.Fatal(err)
	}

	copies[hash] = []byte("a shorter copy")
	played, err := http.Get(answer.Msg.GetMediaUrl())
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

	answer, err := client.GetRecording(t.Context(), connect.NewRequest(&v1.GetRecordingRequest{Path: talk}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetMediaType() != "audio/mpeg" {
		t.Errorf("a recording is played as %q, want audio/mpeg", answer.Msg.GetMediaType())
	}
}
