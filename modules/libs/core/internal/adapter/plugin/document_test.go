package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

type mockDocumentServer struct {
	numenv1connect.UnimplementedDocumentServiceHandler
	lastOpenedPath string
}

func (s *mockDocumentServer) GetDocument(
	_ context.Context,
	req *connect.Request[v1.GetDocumentRequest],
) (*connect.Response[v1.GetDocumentResponse], error) {
	s.lastOpenedPath = req.Msg.GetPath()
	return connect.NewResponse(&v1.GetDocumentResponse{
		Pages: []*v1.Page{
			{Width: 612, Height: 792},
		},
	}), nil
}

func (s *mockDocumentServer) RenderPage(
	_ context.Context,
	_ *connect.Request[v1.RenderPageRequest],
) (*connect.Response[v1.RenderPageResponse], error) {
	return connect.NewResponse(&v1.RenderPageResponse{}), nil
}

func (s *mockDocumentServer) ExtractText(
	_ context.Context,
	_ *connect.Request[v1.ExtractTextRequest],
) (*connect.Response[v1.ExtractTextResponse], error) {
	return connect.NewResponse(&v1.ExtractTextResponse{
		Text:       "sample document text",
		PageStarts: []int32{0},
	}), nil
}

func createTestDocumentPluginTree(t *testing.T, dir string) {
	t.Helper()
	pDir := filepath.Join(dir, "document")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := `{
		"id": "md.numen.document",
		"name": "Document Plugin",
		"version": "1.0.0",
		"capabilities": ["document"],
		"targets": ["desktop.editor"],
		"permissions": ["vault:read"],
		"lifecycle": {
			"mode": "on-demand",
			"idleTimeoutSeconds": 1,
			"keepAliveWhileVisible": false
		},
		"backend": {
			"type": "process",
			"entrypoint": "bin/document",
			"serviceName": "numen.v1.DocumentService"
		}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
}

type unnammedReadSeekCloser struct {
	*bytes.Reader
}

func (u *unnammedReadSeekCloser) Close() error {
	return nil
}

func TestDocumentBridge_PreservesTempFileUntilClose(t *testing.T) {
	mockServer := &mockDocumentServer{}
	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewDocumentServiceHandler(mockServer))

	server := httptest.NewServer(mux)
	defer server.Close()

	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	tmp := t.TempDir()
	createTestDocumentPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
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

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	bridge, err := OpenDocumentBridge(ctx, host)
	if err != nil {
		t.Fatalf("OpenDocumentBridge failed: %v", err)
	}

	// Supply an unnamed file stream to trigger buffering in temp file
	file := &unnammedReadSeekCloser{Reader: bytes.NewReader([]byte("%PDF-dummy-content"))}
	doc, err := bridge.OpenDocument(ctx, file)
	if err != nil {
		t.Fatalf("OpenDocument failed: %v", err)
	}

	tempPath := mockServer.lastOpenedPath
	if tempPath == "" {
		t.Fatal("expected temp document path to be recorded, got empty")
	}

	// Verify temporary file still exists while document is open
	if _, err := os.Stat(tempPath); err != nil {
		t.Fatalf("temporary document file should exist while open: %v", err)
	}

	// Host supervisor should be active while document is open
	host.mu.RLock()
	sup, ok := host.supervisors["md.numen.document"]
	host.mu.RUnlock()
	if !ok {
		t.Fatalf("supervisor for document plugin not found")
	}
	if sup.IsIdle(0, time.Now()) {
		t.Fatalf("supervisor should be active while document is open")
	}

	// Close document and verify temp file is deleted and supervisor is released
	doc.Close()

	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("temporary document file should be removed after Close(), got stat err: %v", err)
	}

	// Also release bridge reference
	_ = bridge.Close()

	if !sup.IsIdle(0, time.Now()) {
		t.Fatalf("supervisor should be idle after document and bridge Close()")
	}
}

func TestDocumentBridgeNoPlugin(t *testing.T) {
	tmp := t.TempDir()
	host := NewHost("desktop.editor", tmp)
	_ = host.Discover()

	_, err := OpenDocumentBridge(t.Context(), host)
	var missing *pluginport.UnavailableError
	if !errors.As(err, &missing) || missing.Reason != pluginport.PluginNotInstalled || missing.Capability != CapabilityDocument {
		t.Fatalf("got %v, want the document plugin not installed", err)
	}
}

func TestRemoteDocumentRenderPageRejectsOutOfRangeInputs(t *testing.T) {
	// The zero service panics on any call, so a rejected input never reaches it.
	doc := &remoteDocument{}
	overflowVal := int64(math.MaxInt32) + 1
	cases := []struct {
		name       string
		index, dpi int
	}{
		{"negative page", -1, 150},
		{"negative dpi", 0, -1},
		{"page above MaxInt32", int(overflowVal), 150},
		{"dpi above MaxInt32", 0, int(overflowVal)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := doc.RenderPage(tc.index, tc.dpi); err == nil {
				t.Fatal("RenderPage returned nil error")
			}
		})
	}
}
