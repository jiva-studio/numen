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

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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

// ExtractBookText answers by the bytes it is given: the bytes are the text, and the
// word "locked" or "junk" names the refusal.
func (s *fakeBookServer) ExtractBookText(
	_ context.Context,
	req *connect.Request[v1.ExtractBookTextRequest],
) (*connect.Response[v1.ExtractBookTextResponse], error) {
	data := string(req.Msg.GetData())
	switch data {
	case "locked":
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("locked"))
	case "junk":
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("junk"))
	}
	return connect.NewResponse(&v1.ExtractBookTextResponse{
		Text:         data,
		Parts:        []*v1.BookPart{{Title: "One", Offset: 0}},
		PrintedPages: []*v1.PrintedPage{{Label: "iv", Offset: 2}},
	}), nil
}

func (s *fakeBookServer) ReadBookEntry(
	_ context.Context,
	req *connect.Request[v1.ReadBookEntryRequest],
) (*connect.Response[v1.ReadBookEntryResponse], error) {
	switch req.Msg.GetEntry() {
	case "changed":
		return nil, connect.NewError(connect.CodeAborted, errors.New("changed"))
	case "missing":
		return nil, connect.NewError(connect.CodeNotFound, errors.New("missing"))
	case "boom":
		return nil, connect.NewError(connect.CodeInternal, errors.New("boom"))
	}
	return connect.NewResponse(&v1.ReadBookEntryResponse{
		Body: []byte(req.Msg.GetPath() + ":" + req.Msg.GetEntry()),
	}), nil
}

func TestBookBridgeWithoutPlugin(t *testing.T) {
	host := NewHost("desktop.editor")
	ctx := t.Context()

	_, err := OpenBookBridge(ctx, host)
	var missing *port.UnavailableError
	if !errors.As(err, &missing) || missing.Reason != port.PluginNotInstalled || missing.Capability != CapabilityBook {
		t.Fatalf("got %v, want the book plugin not installed", err)
	}
}

func TestBookBridgeWithFakePlugin(t *testing.T) {
	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
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

// openFakeBookBridge opens a bridge to a fake book plugin served in this process.
func openFakeBookBridge(t *testing.T, permissions ...string) *BookBridge {
	t.Helper()
	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewBookServiceHandler(&fakeBookServer{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	listenPort := server.Listener.Addr().(*net.TCPAddr).Port

	host := NewHost("desktop.editor")
	p := DiscoveredPlugin{
		Manifest: Manifest{
			ID:           "md.numen.epub",
			Name:         "EPUB Reader",
			Version:      "1.0.0",
			Capabilities: []string{"book"},
			Targets:      []string{"desktop.editor"},
			Permissions:  permissions,
			Backend:      Backend{Entrypoint: "bin/reader-epub"},
		},
		Dir: t.TempDir(),
	}
	host.plugins[p.Manifest.ID] = p
	sup := NewSupervisor()
	sup.SetPermissions(permissions)
	host.supervisors[p.Manifest.ID] = sup
	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r, w := io.Pipe()
		go func() { _, _ = fmt.Fprintf(w, "NUMEN_READY port=%d\n", listenPort) }()
		return &os.Process{Pid: 12345}, r, nil
	})

	bridge, err := OpenBookBridge(t.Context(), host)
	if err != nil {
		t.Fatalf("open the bridge: %v", err)
	}
	t.Cleanup(func() { _ = bridge.Close() })
	return bridge
}

func TestBookBridgeReadBookAnswersTextPartsAndPages(t *testing.T) {
	bridge := openFakeBookBridge(t, PermissionVaultRead)
	got, err := bridge.ReadBook(t.Context(), []byte("some words"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "some words" {
		t.Errorf("text = %q, want the text the plugin answered", got.Text)
	}
	if len(got.Parts) != 1 || got.Parts[0].Title != "One" || got.Parts[0].Offset != 0 {
		t.Errorf("parts = %+v, want the part the plugin named", got.Parts)
	}
	if len(got.Pages) != 1 || got.Pages[0].Label != "iv" || got.Pages[0].Offset != 2 {
		t.Errorf("pages = %+v, want the printed page the plugin named", got.Pages)
	}
}

func TestBookBridgeReadBookNamesWhyABookIsRefused(t *testing.T) {
	bridge := openFakeBookBridge(t, PermissionVaultRead)
	if _, err := bridge.ReadBook(t.Context(), []byte("junk")); !errors.Is(err, port.ErrNotADocument) {
		t.Errorf("bytes that are no book: err = %v, want ErrNotADocument", err)
	}
	if _, err := bridge.ReadBook(t.Context(), []byte("locked")); !errors.Is(err, port.ErrEncrypted) {
		t.Errorf("a locked book: err = %v, want ErrEncrypted", err)
	}
}

func TestBookBridgeReadEntryAsksForTheEntryOfTheFile(t *testing.T) {
	bridge := openFakeBookBridge(t, PermissionVaultRead)
	body, err := bridge.ReadEntry(t.Context(), &v1.Fingerprint{Path: "a.epub", Size: 1, Mtime: 2}, "cover.png")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "a.epub:cover.png" {
		t.Errorf("body = %q, want the entry of the file asked for", body)
	}
}

func TestBookBridgeReadEntryNamesWhyAnEntryIsRefused(t *testing.T) {
	bridge := openFakeBookBridge(t, PermissionVaultRead)
	named := &v1.Fingerprint{Path: "a.epub"}
	if _, err := bridge.ReadEntry(t.Context(), named, "changed"); !errors.Is(err, port.ErrBookChanged) {
		t.Errorf("a file that changed: err = %v, want ErrBookChanged", err)
	}
	if _, err := bridge.ReadEntry(t.Context(), named, "missing"); !errors.Is(err, port.ErrNoBookEntry) {
		t.Errorf("an entry the archive lacks: err = %v, want ErrNoBookEntry", err)
	}
	_, err := bridge.ReadEntry(t.Context(), named, "boom")
	if err == nil || errors.Is(err, port.ErrNoBookEntry) || errors.Is(err, port.ErrBookChanged) {
		t.Errorf("a failed plugin: err = %v, want an error that is none of the book's own", err)
	}
}

func TestBookBridgeRefusesAPluginWithoutVaultRead(t *testing.T) {
	bridge := openFakeBookBridge(t)
	if _, err := bridge.ReadBook(t.Context(), []byte("x")); err == nil {
		t.Error("ReadBook was answered for a plugin without vault read")
	}
	if _, err := bridge.ReadEntry(t.Context(), &v1.Fingerprint{Path: "a"}, "e"); err == nil {
		t.Error("ReadEntry was answered for a plugin without vault read")
	}
}
