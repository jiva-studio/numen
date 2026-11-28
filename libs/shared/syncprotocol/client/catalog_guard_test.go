package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

type pagedCatalog struct {
	syncv1connect.UnimplementedCatalogServiceHandler
	calls   atomic.Int64
	advance func(call int64, since uint64) []*syncv1.CatalogEntry
}

func (s *pagedCatalog) GetCatalog(_ context.Context, req *connect.Request[syncv1.GetCatalogRequest]) (*connect.Response[syncv1.GetCatalogResponse], error) {
	call := s.calls.Add(1)
	return connect.NewResponse(&syncv1.GetCatalogResponse{Entries: s.advance(call, req.Msg.GetSinceVersion()), HasMore: true}), nil
}

func serveCatalog(t *testing.T, advance func(call int64, since uint64) []*syncv1.CatalogEntry) (*pagedCatalog, *syncclient.CatalogClient) {
	t.Helper()
	stub := &pagedCatalog{advance: advance}
	mux := http.NewServeMux()
	mux.Handle(syncv1connect.NewCatalogServiceHandler(stub, connect.WithInterceptors(syncprotocol.NewVersionHandlerInterceptor(syncprotocol.Version))))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return stub, syncclient.NewCatalogClient(server.URL, http.DefaultClient, func() string { return "t" })
}

func TestGetCatalog_APageThatDoesNotAdvanceTheVersionEndsTheRead(t *testing.T) {
	t.Parallel()
	stub, catalog := serveCatalog(t, func(_ int64, since uint64) []*syncv1.CatalogEntry {
		return []*syncv1.CatalogEntry{{Version: since}}
	})

	_, err := catalog.GetCatalog(t.Context(), 7)

	if err == nil {
		t.Fatal("a server that repeats a version was believed")
	}
	if got := stub.calls.Load(); got > 2 {
		t.Errorf("the read asked %d times, want it to stop at the first page that did not advance", got)
	}
}

func TestGetCatalog_AServerThatNeverEndsIsStoppedAtTheEntryLimit(t *testing.T) {
	t.Parallel()
	stub, catalog := serveCatalog(t, func(_ int64, since uint64) []*syncv1.CatalogEntry {
		entries := make([]*syncv1.CatalogEntry, syncprotocol.MaxPageSize)
		for i := range entries {
			entries[i] = &syncv1.CatalogEntry{Version: since + uint64(i) + 1}
		}
		return entries
	})

	entries, err := catalog.GetCatalog(t.Context(), 0)

	if err == nil || len(entries) != 0 {
		t.Fatalf("got %d entries and %v, want the limit as an error", len(entries), err)
	}
	if got := stub.calls.Load(); got > int64(syncprotocol.MaxCatalogEntries/syncprotocol.MaxPageSize)+1 {
		t.Errorf("the read asked %d times", got)
	}
}
