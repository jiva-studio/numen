package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type fakeBookServer struct {
	numenv1connect.UnimplementedBookServiceHandler
}

func (s *fakeBookServer) GetBook(
	_ context.Context,
	req *connect.Request[v1.GetBookRequest],
) (*connect.Response[v1.GetBookResponse], error) {
	if req.Msg.GetPath() == "nonexistent.epub" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("not found"))
	}
	return connect.NewResponse(&v1.GetBookResponse{
		Title:     "Bridged Book",
		PageCount: 10,
		TextBytes: 5000,
	}), nil
}

func (s *fakeBookServer) ReadBookMarkup(
	_ context.Context,
	_ *connect.Request[v1.ReadBookMarkupRequest],
) (*connect.Response[v1.ReadBookMarkupResponse], error) {
	return connect.NewResponse(&v1.ReadBookMarkupResponse{
		Markup: "<p>Bridged markup</p>",
	}), nil
}

func TestBookBridgeWithoutPlugin(t *testing.T) {
	host := NewHost("desktop.editor")
	ctx := t.Context()

	_, err := OpenBookBridge(ctx, host)
	if !errors.Is(err, ErrNoBookPlugin) {
		t.Fatalf("expected ErrNoBookPlugin, got %v", err)
	}
}

func TestBookBridgeWithFakePlugin(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewBookServiceHandler(&fakeBookServer{}))
	server := httptest.NewServer(mux)
	defer server.Close()

	port := server.Listener.Addr().(*net.TCPAddr).Port

	host := NewHost("desktop.editor")
	p := DiscoveredPlugin{
		Manifest: Manifest{
			ID:           "md.numen.epub",
			Name:         "EPUB Reader",
			Version:      "1.0.0",
			Capabilities: []string{"book"},
			Targets:      []string{"desktop.editor"},
			Backend:      Backend{Entrypoint: "bin/reader-epub"},
		},
		Dir: t.TempDir(),
	}
	host.plugins[p.Manifest.ID] = p
	host.supervisors[p.Manifest.ID] = NewSupervisor()

	fakeLauncher := func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r, w := io.Pipe()
		go func() {
			_, _ = fmt.Fprintf(w, "NUMEN_READY port=%d\n", port)
		}()
		return &os.Process{Pid: 12345}, r, nil
	}
	host.SetLauncher(fakeLauncher)

	ctx := t.Context()
	bridge, err := OpenBookBridge(ctx, host)
	if err != nil {
		t.Fatalf("unexpected OpenBookBridge error: %v", err)
	}
	defer bridge.Close()

	// Test GetBook
	resp, err := bridge.GetBook(ctx, connect.NewRequest(&v1.GetBookRequest{
		Path: "test.epub",
	}))
	if err != nil {
		t.Fatalf("unexpected GetBook error: %v", err)
	}
	if resp.Msg.GetTitle() != "Bridged Book" {
		t.Fatalf("expected title 'Bridged Book', got %q", resp.Msg.GetTitle())
	}

	// Test ReadBookMarkup
	markupResp, err := bridge.ReadBookMarkup(ctx, connect.NewRequest(&v1.ReadBookMarkupRequest{
		Path:     "test.epub",
		Document: "doc.xhtml",
	}))
	if err != nil {
		t.Fatalf("unexpected ReadBookMarkup error: %v", err)
	}
	if !strings.Contains(markupResp.Msg.GetMarkup(), "Bridged markup") {
		t.Fatalf("unexpected markup: %s", markupResp.Msg.GetMarkup())
	}
}
