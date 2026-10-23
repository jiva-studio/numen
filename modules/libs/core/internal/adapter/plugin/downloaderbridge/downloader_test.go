package downloaderbridge

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/downloadport"
)

type mockDownloadServer struct{}

func (s *mockDownloadServer) CanHandle(
	_ context.Context,
	_ *connect.Request[v1.CanHandleRequest],
) (*connect.Response[v1.CanHandleResponse], error) {
	return connect.NewResponse(&v1.CanHandleResponse{
		CanHandle: true,
	}), nil
}

func (s *mockDownloadServer) GetDownloadModel(
	_ context.Context,
	_ *connect.Request[v1.GetDownloadModelRequest],
) (*connect.Response[v1.GetDownloadModelResponse], error) {
	return connect.NewResponse(&v1.GetDownloadModelResponse{
		Tool:     "test-ytdlp",
		Version:  "2026.01.01",
		Producer: "yt-dlp",
	}), nil
}

func (s *mockDownloadServer) GetMetadata(
	_ context.Context,
	_ *connect.Request[v1.GetMetadataRequest],
) (*connect.Response[v1.GetMetadataResponse], error) {
	return connect.NewResponse(&v1.GetMetadataResponse{
		Title:       "Test Video",
		Bytes:       2048,
		Length:      60000,
		Language:    "en",
		Captions:    []string{"en"},
		IsAutomatic: []string{"de"},
	}), nil
}

func (s *mockDownloadServer) GetText(
	_ context.Context,
	_ *connect.Request[v1.GetTextRequest],
) (*connect.Response[v1.GetTextResponse], error) {
	return connect.NewResponse(&v1.GetTextResponse{
		Producer: "yt-dlp",
		Cues: []*v1.Cue{
			{Text: "Hello world", From: 0, To: 2000},
		},
		Prose:  "Hello world",
		Title:  "Test Video",
		Length: 60000,
	}), nil
}

func (s *mockDownloadServer) DownloadMedia(
	_ context.Context,
	_ *connect.Request[v1.DownloadMediaRequest],
	stream *connect.ServerStream[v1.DownloadMediaResponse],
) error {
	if err := stream.Send(&v1.DownloadMediaResponse{
		Chunk: []byte("media-bytes-payload"),
	}); err != nil {
		return err
	}
	return stream.Send(&v1.DownloadMediaResponse{
		MediaType: "video/mp4",
		Extension: ".mp4",
	})
}

func createTestDownloaderPluginTree(t *testing.T, dir string) {
	t.Helper()
	pDir := filepath.Join(dir, "downloader")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := `{
		"id": "md.numen.downloader",
		"name": "Media Downloader Plugin",
		"version": "1.0.0",
		"description": "Media Downloader",
		"capabilities": ["downloader"],
		"targets": ["desktop.editor"],
		"permissions": ["network:outbound", "vault:write"],
		"lifecycle": {
			"mode": "on-demand",
			"idleTimeoutSeconds": 1,
			"keepAliveWhileVisible": false
		},
		"backend": {
			"type": "process",
			"entrypoint": "bin/downloader",
			"serviceName": "numen.v1.DownloadService"
		}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
}

func TestDownloaderBridge(t *testing.T) {
	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewDownloadServiceHandler(&mockDownloadServer{}))

	server := httptest.NewServer(mux)
	defer server.Close()

	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	tmp := t.TempDir()
	createTestDownloaderPluginTree(t, tmp)

	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ plugindir.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	bridge, err := Open(ctx, host)
	if err != nil {
		t.Fatalf("OpenDownloaderBridge failed: %v", err)
	}
	defer bridge.Close()

	url := domain.URL("https://example.com/video")
	can, err := bridge.CanHandle(ctx, url)
	if err != nil || !can {
		t.Fatalf("CanHandle failed: %v, can=%v", err, can)
	}

	model, err := bridge.GetDownloadModel(ctx, url)
	if err != nil {
		t.Fatalf("GetDownloadModel failed: %v", err)
	}
	if model.Tool != "test-ytdlp" || model.Version != "2026.01.01" || model.Producer != "yt-dlp" {
		t.Fatalf("unexpected model: %+v", model)
	}

	meta, err := bridge.GetMetadata(ctx, url)
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}
	if meta.Title != "Test Video" || meta.Bytes != 2048 || meta.Length != 60000 {
		t.Fatalf("unexpected metadata: %+v", meta)
	}

	text, err := bridge.GetText(ctx, url, downloadport.PreferredCaptions{
		Languages: []string{"en"},
	})
	if err != nil {
		t.Fatalf("GetText failed: %v", err)
	}
	if text.Producer != "yt-dlp" || len(text.Cues) != 1 || text.Cues[0].Text != "Hello world" {
		t.Fatalf("unexpected text: %+v", text)
	}

	var buf bytes.Buffer
	copyInfo, err := bridge.Download(ctx, url, &buf)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if copyInfo.MediaType != "video/mp4" || copyInfo.Extension != ".mp4" {
		t.Fatalf("unexpected copy: %+v", copyInfo)
	}
	if buf.String() != "media-bytes-payload" {
		t.Fatalf("unexpected buffer: %s", buf.String())
	}
}

func TestDownloaderBridgeNoPlugin(t *testing.T) {
	tmp := t.TempDir()
	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	_, err := Open(t.Context(), host)
	if err == nil {
		t.Fatal("expected error when no downloader plugin registered, got nil")
	}
}
