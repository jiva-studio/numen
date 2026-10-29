package host_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// storeHost is a host whose plugin folder is a map, and that records the vault each call named.
type storeHost struct {
	numenv1connect.UnimplementedHostStoreServiceHandler
	mu     sync.Mutex
	files  map[string][]byte
	code   v1.HostStoreErrorCode
	vaults []string
}

func newStoreHost() *storeHost {
	return &storeHost{files: map[string][]byte{}}
}

func (h *storeHost) recordVault(header http.Header) {
	h.vaults = append(h.vaults, header.Get("Numen-Vault"))
}

func (h *storeHost) ReadStoreFile(_ context.Context, req *connect.Request[v1.ReadStoreFileRequest]) (*connect.Response[v1.ReadStoreFileResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recordVault(req.Header())
	if h.code != 0 {
		return connect.NewResponse(&v1.ReadStoreFileResponse{Error: h.code}), nil
	}
	content, there := h.files[req.Msg.GetName()]
	if !there {
		return connect.NewResponse(&v1.ReadStoreFileResponse{Error: v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_MISSING}), nil
	}
	return connect.NewResponse(&v1.ReadStoreFileResponse{Content: content}), nil
}

func (h *storeHost) AppendStoreFile(_ context.Context, req *connect.Request[v1.AppendStoreFileRequest]) (*connect.Response[v1.AppendStoreFileResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recordVault(req.Header())
	h.files[req.Msg.GetName()] = append(h.files[req.Msg.GetName()], req.Msg.GetContent()...)
	return connect.NewResponse(&v1.AppendStoreFileResponse{}), nil
}

func (h *storeHost) ListStoreFiles(_ context.Context, req *connect.Request[v1.ListStoreFilesRequest]) (*connect.Response[v1.ListStoreFilesResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recordVault(req.Header())
	var files []*v1.HostStoreFile
	for name, content := range h.files {
		files = append(files, &v1.HostStoreFile{Name: name, Size: int64(len(content))})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].GetName() < files[j].GetName() })
	return connect.NewResponse(&v1.ListStoreFilesResponse{Files: files}), nil
}

func (h *storeHost) WriteStoreFile(_ context.Context, req *connect.Request[v1.WriteStoreFileRequest]) (*connect.Response[v1.WriteStoreFileResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recordVault(req.Header())
	h.files[req.Msg.GetName()] = req.Msg.GetContent()
	return connect.NewResponse(&v1.WriteStoreFileResponse{}), nil
}

func (h *storeHost) DeleteStoreFile(_ context.Context, req *connect.Request[v1.DeleteStoreFileRequest]) (*connect.Response[v1.DeleteStoreFileResponse], error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recordVault(req.Header())
	delete(h.files, req.Msg.GetName())
	return connect.NewResponse(&v1.DeleteStoreFileResponse{Error: h.code}), nil
}

func dialStore(t *testing.T, h *storeHost) *host.StoreClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostStoreServiceHandler(h))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return host.NewStoreClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestAStoreClientAppendsReadsAndLists(t *testing.T) {
	ctx := context.Background()
	c := dialStore(t, newStoreHost())

	first := c.Append(ctx, "a", []byte("one\n"))
	second := c.Append(ctx, "a", []byte("two\n"))
	got, readErr := c.Read(ctx, "a")
	listed, listErr := c.List(ctx)
	_, missing := c.Read(ctx, "none")

	if first != nil || second != nil || readErr != nil || string(got) != "one\ntwo\n" {
		t.Errorf("got %q (%v, %v, %v)", got, first, second, readErr)
	}
	if listErr != nil || len(listed) != 1 || listed[0] != (host.StoreFile{Name: "a", Size: 8}) {
		t.Errorf("listed %v (%v)", listed, listErr)
	}
	if !errors.Is(missing, host.ErrNotExist) {
		t.Errorf("a missing file was read with %v", missing)
	}
}

func TestEachCodeOfTheStoreIsItsOwnError(t *testing.T) {
	cases := map[v1.HostStoreErrorCode]error{
		v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_NO_VAULT:        host.ErrNoVault,
		v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_MISSING:         host.ErrNotExist,
		v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_OUTSIDE:         host.ErrOutside,
		v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_HELD_BY_ANOTHER: host.ErrLockedByAnother,
	}
	for code, want := range cases {
		h := newStoreHost()
		h.code = code
		if _, err := dialStore(t, h).Read(context.Background(), "a"); !errors.Is(err, want) {
			t.Errorf("code %s: got %v, want %v", code, err, want)
		}
	}
}

func TestAStoreOfAnotherVaultNamesItOnEveryCall(t *testing.T) {
	ctx := context.Background()
	h := newStoreHost()
	c := dialStore(t, h)

	_, _ = c.With("Chemistry").Read(ctx, "a")
	_ = c.With("Chemistry").Append(ctx, "a", []byte("x"))
	_, _ = c.With("Chemistry").List(ctx)
	_, _ = c.Read(ctx, "a")

	want := []string{"Chemistry", "Chemistry", "Chemistry", ""}
	if len(h.vaults) != len(want) {
		t.Fatalf("the vaults named were %q", h.vaults)
	}
	for i := range want {
		if h.vaults[i] != want[i] {
			t.Errorf("the vaults named were %q, want %q", h.vaults, want)
		}
	}
}

func TestStoreClientWriteAndDelete(t *testing.T) {
	h := newStoreHost()
	store := dialStore(t, h)
	ctx := t.Context()

	if err := store.Write(ctx, "a.json", []byte("first and longer")); err != nil {
		t.Fatal(err)
	}
	if err := store.Write(ctx, "a.json", []byte("second")); err != nil {
		t.Fatal(err)
	}
	read, err := store.Read(ctx, "a.json")
	if err != nil || string(read) != "second" {
		t.Errorf("read %q (%v), want the second write alone", read, err)
	}
	if err := store.Delete(ctx, "a.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Read(ctx, "a.json"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("a deleted file was read: %v", err)
	}
	h.code = v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_OUTSIDE
	if err := store.Delete(ctx, "../a"); !errors.Is(err, host.ErrOutside) {
		t.Errorf("a refused delete answered %v", err)
	}
}
