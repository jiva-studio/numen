package container

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

type bookServer struct {
	numenv1connect.UnimplementedBookServiceHandler
}

func (bookServer) GetBook(
	context.Context, *connect.Request[v1.GetBookRequest],
) (*connect.Response[v1.GetBookResponse], error) {
	return connect.NewResponse(&v1.GetBookResponse{Title: "A Book"}), nil
}

func (bookServer) ReadBookEntry(
	context.Context, *connect.Request[v1.ReadBookEntryRequest],
) (*connect.Response[v1.ReadBookEntryResponse], error) {
	return nil, connect.NewError(connect.CodeNotFound, errors.New("no such entry"))
}

// A book asked of an installation with no plugin that reads books is refused as
// unavailable, whatever the ask is.
func TestBooksAreUnavailableWhereNoPluginReadsThem(t *testing.T) {
	for name, cfg := range map[string]Config{
		"no plugin host": {},
		"safe mode":      {PluginHost: &countingDocumentHost{}, SafeMode: true},
	} {
		t.Run(name, func(t *testing.T) {
			books := cfg.Books()
			if _, err := books.GetBook(t.Context(), "a.epub"); !errors.Is(err, port.ErrUnavailable) {
				t.Errorf("GetBook: err = %v, want unavailable", err)
			}
			if _, err := books.ReadBookMarkup(t.Context(), "a.epub", "one.xhtml", nil); !errors.Is(err, port.ErrUnavailable) {
				t.Errorf("ReadBookMarkup: err = %v, want unavailable", err)
			}
			if _, err := books.ReadEntry(t.Context(), domain.Fingerprint{Path: "a.epub"}, "cover.png"); !errors.Is(err, port.ErrUnavailable) {
				t.Errorf("ReadEntry: err = %v, want unavailable", err)
			}
		})
	}
}

// A book is read through the plugin, an entry it lacks is the port's own refusal,
// and every reference taken for a call is given back.
func TestBooksAreReadThroughThePluginAndGiveBackEveryReference(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewBookServiceHandler(bookServer{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	host := &countingDocumentHost{endpoint: port.Endpoint{Host: "127.0.0.1", Port: server.Listener.Addr().(*net.TCPAddr).Port}}
	books := Config{PluginHost: host}.Books()

	shape, err := books.GetBook(t.Context(), "a.epub")
	if err != nil || shape.Title != "A Book" {
		t.Fatalf("shape = %+v, err = %v", shape, err)
	}
	if _, err := books.ReadEntry(t.Context(), domain.Fingerprint{Path: "a.epub"}, "cover.png"); !errors.Is(err, port.ErrNoBookEntry) {
		t.Errorf("ReadEntry: err = %v, want ErrNoBookEntry", err)
	}
	if host.acquired.Load() != host.released.Load() {
		t.Errorf("acquired %d, released %d", host.acquired.Load(), host.released.Load())
	}
}
