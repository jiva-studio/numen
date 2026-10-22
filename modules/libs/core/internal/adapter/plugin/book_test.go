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
	"reflect"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
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
	if req.Msg.GetPath() == "locked.epub" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("locked"))
	}
	return connect.NewResponse(&v1.GetBookResponse{
		Title:        "Bridged Book",
		Documents:    []*v1.SpineDocument{{Path: "one.xhtml", Offset: 1, Length: 9}},
		Parts:        []*v1.BookPart{{Title: "One", Offset: 1, Level: 2}},
		PrintedPages: []*v1.PrintedPage{{Label: "iv", Offset: 4}},
		PageCount:    10,
		TextBytes:    5000,
		PageBytes:    500,
		Fingerprint:  &v1.Fingerprint{Path: req.Msg.GetPath(), Size: 12, Mtime: 34},
	}), nil
}

func (s *fakeBookServer) ReadBookMarkup(
	_ context.Context,
	req *connect.Request[v1.ReadBookMarkupRequest],
) (*connect.Response[v1.ReadBookMarkupResponse], error) {
	switch req.Msg.GetDocument() {
	case "changed.xhtml":
		return nil, connect.NewError(connect.CodeAborted, errors.New("changed"))
	case "missing.xhtml":
		return nil, connect.NewError(connect.CodeNotFound, errors.New("missing"))
	case "baseline.xhtml":
		mark := req.Msg.GetBaseline()
		return connect.NewResponse(&v1.ReadBookMarkupResponse{
			Markup: fmt.Sprintf("%t:%s:%d:%d", mark != nil, mark.GetPath(), mark.GetSize(), mark.GetMtime()),
		}), nil
	}
	return connect.NewResponse(&v1.ReadBookMarkupResponse{
		Markup: "<p>Bridged markup</p>",
	}), nil
}

// ExtractBookText answers by the bytes it is given: the bytes are the text, and the
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
			ID:           "md.numen.readers.epub",
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
	shape, err := bridge.GetBook(ctx, "test.epub")
	if err != nil {
		t.Fatalf("unexpected GetBook error: %v", err)
	}
	if shape.Title != "Bridged Book" {
		t.Fatalf("expected title 'Bridged Book', got %q", shape.Title)
	}

	// Test ReadBookMarkup
	markup, err := bridge.ReadBookMarkup(ctx, "test.epub", "doc.xhtml", nil)
	if err != nil {
		t.Fatalf("unexpected ReadBookMarkup error: %v", err)
	}
	if !strings.Contains(markup, "Bridged markup") {
		t.Fatalf("unexpected markup: %s", markup)
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
			ID:           "md.numen.readers.epub",
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
	body, err := bridge.ReadEntry(t.Context(), domain.Fingerprint{Path: "a.epub", Size: 1, ModTime: time.Unix(0, 2)}, "cover.png")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "a.epub:cover.png" {
		t.Errorf("body = %q, want the entry of the file asked for", body)
	}
}

func TestBookBridgeReadEntryNamesWhyAnEntryIsRefused(t *testing.T) {
	bridge := openFakeBookBridge(t, PermissionVaultRead)
	named := domain.Fingerprint{Path: "a.epub"}
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
	if _, err := bridge.ReadEntry(t.Context(), domain.Fingerprint{Path: "a"}, "e"); err == nil {
		t.Error("ReadEntry was answered for a plugin without vault read")
	}
}

func TestBookBridgeGetBookAnswersTheShapeOfTheBook(t *testing.T) {
	bridge := openFakeBookBridge(t, PermissionVaultRead)
	got, err := bridge.GetBook(t.Context(), "a.epub")
	if err != nil {
		t.Fatal(err)
	}
	want := domain.BookShape{
		Title:       "Bridged Book",
		Sections:    []domain.BookSection{{Path: "one.xhtml", Offset: 1, Length: 9}},
		Parts:       []domain.BookPart{{Title: "One", Offset: 1, Level: 2}},
		Pages:       []domain.BookPage{{Label: "iv", Offset: 4}},
		PageCount:   10,
		TextBytes:   5000,
		PageBytes:   500,
		Fingerprint: domain.Fingerprint{Path: "a.epub", Size: 12, ModTime: time.Unix(0, 34)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("shape = %+v, want %+v", got, want)
	}
}

func TestBookBridgeGetBookNamesWhyABookIsRefused(t *testing.T) {
	bridge := openFakeBookBridge(t, PermissionVaultRead)
	if _, err := bridge.GetBook(t.Context(), "nonexistent.epub"); !errors.Is(err, port.ErrNoBook) || errors.Is(err, port.ErrNoBookEntry) {
		t.Errorf("a path with no book: err = %v, want ErrNoBook", err)
	}
	if _, err := bridge.GetBook(t.Context(), "locked.epub"); !errors.Is(err, port.ErrEncrypted) {
		t.Errorf("a locked book: err = %v, want ErrEncrypted", err)
	}
}

func TestBookBridgeReadBookMarkupSendsTheBaselineItIsGiven(t *testing.T) {
	bridge := openFakeBookBridge(t, PermissionVaultRead)
	got, err := bridge.ReadBookMarkup(t.Context(), "a.epub", "baseline.xhtml", nil)
	if err != nil || got != "false::0:0" {
		t.Errorf("no baseline: markup = %q, err = %v, want the plugin told none", got, err)
	}
	baseline := &domain.Fingerprint{Path: "a.epub", Size: 3, ModTime: time.Unix(0, 9)}
	got, err = bridge.ReadBookMarkup(t.Context(), "a.epub", "baseline.xhtml", baseline)
	if err != nil || got != "true:a.epub:3:9" {
		t.Errorf("a baseline: markup = %q, err = %v, want the plugin told the baseline", got, err)
	}
}

func TestBookBridgeReadBookMarkupNamesWhyASectionIsRefused(t *testing.T) {
	bridge := openFakeBookBridge(t, PermissionVaultRead)
	if _, err := bridge.ReadBookMarkup(t.Context(), "a.epub", "changed.xhtml", nil); !errors.Is(err, port.ErrBookChanged) {
		t.Errorf("a file that changed: err = %v, want ErrBookChanged", err)
	}
	if _, err := bridge.ReadBookMarkup(t.Context(), "a.epub", "missing.xhtml", nil); !errors.Is(err, port.ErrNoBookEntry) {
		t.Errorf("a section the book lacks: err = %v, want ErrNoBookEntry", err)
	}
}
