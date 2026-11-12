package editor

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func newMediaClient(t *testing.T) numenv1connect.MediaServiceClient {
	t.Helper()
	api, _ := openRecordingWindow(t, nil)
	back, err := Listen(t.Context(), api.server, nil)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, loopbackCloser{back})
	api.storePlayingLoopback(back)
	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	return numenv1connect.NewMediaServiceClient(server.Client(), server.URL)
}

func TestResolveAddress_ReturnsTheLoopbackAddressOfAVaultFile(t *testing.T) {
	client := newMediaClient(t)

	answer, err := client.ResolveAddress(t.Context(), connect.NewRequest(&v1.ResolveAddressRequest{Address: talk}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetMediaType() != "audio/mpeg" {
		t.Errorf("a recording is played as %q, want audio/mpeg", answer.Msg.GetMediaType())
	}
	played := fetch(t, http.DefaultClient, answer.Msg.GetSrc())
	if string(played.Body) != sound {
		t.Errorf("the address served %q, want the recording's bytes", played.Body)
	}
}

func TestResolveAddress_RejectsAMissingVaultFile(t *testing.T) {
	client := newMediaClient(t)

	_, err := client.ResolveAddress(t.Context(), connect.NewRequest(&v1.ResolveAddressRequest{Address: "talks/none.mp3"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("a missing file was answered %v, want not found", err)
	}
}

// storedFiles are the files of the plugins' stores of every vault, by plugin id and then by name.
type storedFiles map[string]map[string][]byte

func (s storedFiles) Open(_ domain.Vault, pluginID string) (pluginport.PluginStoreReader, error) {
	return storedFolder{files: s[pluginID]}, nil
}

func (storedFiles) DeleteFolder(context.Context, domain.Vault, string) error { return nil }

// storedFolder is one plugin's folder, of which a file is opened and nothing else.
type storedFolder struct {
	pluginport.PluginStoreReader
	files map[string][]byte
}

func (f storedFolder) Open(_ context.Context, name string) (io.ReadSeekCloser, int64, error) {
	content, ok := f.files[name]
	if !ok {
		return nil, 0, fs.ErrNotExist
	}
	return nopSeekCloser{bytes.NewReader(content)}, int64(len(content)), nil
}

type nopSeekCloser struct{ io.ReadSeeker }

func (nopSeekCloser) Close() error { return nil }

const (
	importer  = "org.example.links"
	clipName  = "clip-1.mp4"
	clipBytes = "the bytes of a downloaded video, long enough to ask for a piece of"
	pageName  = "clip-1.html"
	pageBytes = "<!doctype html><iframe src=\"https://player.example/embed/1\"></iframe>"
)

func newStoreClient(t *testing.T, files storedFiles) numenv1connect.MediaServiceClient {
	t.Helper()
	vault := vaultfixture.NewVault(t, map[string]string{talk: sound})
	api := newBareServices()
	readers := vaultreader.Readers{}
	api.readFiles(readers, file.NewOpenPluginFile(files), file.NewGetMediaType(contenttype.Open()))
	api.state.Runtime.SetVault(vault)
	back, err := Listen(t.Context(), api.server, nil)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, loopbackCloser{back})
	back.setPluginServices(&routedPlugins{frames: map[string][]string{importer: {"https://player.example"}}})
	api.storePlayingLoopback(back)
	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	return numenv1connect.NewMediaServiceClient(server.Client(), server.URL)
}

func TestResolveAddress_ServesAFileOfAPluginStoreInPieces(t *testing.T) {
	client := newStoreClient(t, storedFiles{importer: {clipName: []byte(clipBytes)}})

	answer, err := client.ResolveAddress(t.Context(), connect.NewRequest(&v1.ResolveAddressRequest{Address: "plugin://" + importer + "/" + clipName}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetMediaType() != "video/mp4" {
		t.Errorf("the file is played as %q, want video/mp4", answer.Msg.GetMediaType())
	}
	asked, err := http.NewRequestWithContext(t.Context(), http.MethodGet, answer.Msg.GetSrc(), nil)
	if err != nil {
		t.Fatal(err)
	}
	asked.Header.Set("Range", "bytes=4-12")
	played := send(t, http.DefaultClient, asked)
	if played.StatusCode != http.StatusPartialContent || string(played.Body) != clipBytes[4:13] {
		t.Errorf("the address answered %d with %q, want a piece of the file", played.StatusCode, played.Body)
	}
}

func TestResolveAddress_ServesAPageOfAPluginStoreWithItsFramesAndReferrer(t *testing.T) {
	client := newStoreClient(t, storedFiles{importer: {pageName: []byte(pageBytes)}})

	answer, err := client.ResolveAddress(t.Context(), connect.NewRequest(&v1.ResolveAddressRequest{Address: "plugin://" + importer + "/" + pageName}))
	if err != nil {
		t.Fatal(err)
	}
	resp := fetch(t, http.DefaultClient, answer.Msg.GetSrc())
	if resp.StatusCode != http.StatusOK || string(resp.Body) != pageBytes {
		t.Fatalf("answered %d %q", resp.StatusCode, resp.Body)
	}
	if policy := resp.Header.Get("Content-Security-Policy"); !strings.Contains(policy, "frame-src 'self' https://player.example;") {
		t.Errorf("the page is served under %q", policy)
	}
	if referrer := resp.Header.Get("Referrer-Policy"); referrer != "strict-origin-when-cross-origin" {
		t.Errorf("the page sends Referrer-Policy %q", referrer)
	}
}

func TestResolveAddress_RejectsTheAddressOfAPluginFileOfAnotherSize(t *testing.T) {
	files := storedFiles{importer: {clipName: []byte(clipBytes)}}
	client := newStoreClient(t, files)
	answer, err := client.ResolveAddress(t.Context(), connect.NewRequest(&v1.ResolveAddressRequest{Address: "plugin://" + importer + "/" + clipName}))
	if err != nil {
		t.Fatal(err)
	}

	files[importer][clipName] = []byte("a shorter video")
	played := fetch(t, http.DefaultClient, answer.Msg.GetSrc())
	if played.StatusCode != http.StatusNotFound {
		t.Errorf("an address of the file before was answered %d", played.StatusCode)
	}
}

func TestResolveAddress_AnswersAMissingPluginFileAndAMalformedAddress(t *testing.T) {
	client := newStoreClient(t, storedFiles{importer: {}})
	for address, want := range map[string]connect.Code{
		"plugin://" + importer + "/gone.mp4": connect.CodeNotFound,
		"plugin://" + importer:               connect.CodeInvalidArgument,
		"plugin:///" + clipName:              connect.CodeInvalidArgument,
	} {
		_, err := client.ResolveAddress(t.Context(), connect.NewRequest(&v1.ResolveAddressRequest{Address: address}))
		if connect.CodeOf(err) != want {
			t.Errorf("%s was answered %v, want %v", address, err, want)
		}
	}
}
