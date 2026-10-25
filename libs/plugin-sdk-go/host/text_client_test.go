package host_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// textHost answers HostTextService with what a test set, and records what it was put.
type textHost struct {
	numenv1connect.UnimplementedHostTextServiceHandler
	code    v1.TextErrorCode
	request *v1.PutTextVersionRequest
	auth    string
}

func (h *textHost) PutTextVersion(_ context.Context, req *connect.Request[v1.PutTextVersionRequest]) (*connect.Response[v1.PutTextVersionResponse], error) {
	h.request, h.auth = req.Msg, req.Header().Get("Authorization")
	if h.code != v1.TextErrorCode_TEXT_ERROR_CODE_UNSPECIFIED {
		return connect.NewResponse(&v1.PutTextVersionResponse{Error: h.code}), nil
	}
	return connect.NewResponse(&v1.PutTextVersionResponse{Version: &v1.TextVersion{
		Id: "p/" + req.Msg.GetKey(), Path: req.Msg.GetPath(), PluginId: "p", Key: req.Msg.GetKey(),
		Format: req.Msg.GetFormat(), CreateTimeUnixMs: 1_700_000_000_000,
	}}), nil
}

func (h *textHost) ListTextVersions(context.Context, *connect.Request[v1.ListTextVersionsRequest]) (*connect.Response[v1.ListTextVersionsResponse], error) {
	return connect.NewResponse(&v1.ListTextVersionsResponse{
		Versions:      []*v1.TextVersion{{Id: "p/k", Path: "a.pdf", PluginId: "p", Key: "k", Format: v1.TextFormat_TEXT_FORMAT_TIMED}},
		LiveVersionId: "p/k",
		Error:         h.code,
	}), nil
}

func (h *textHost) ReadText(_ context.Context, req *connect.Request[v1.ReadTextRequest]) (*connect.Response[v1.ReadTextResponse], error) {
	if req.Msg.GetCursor() == "" {
		return connect.NewResponse(&v1.ReadTextResponse{Parts: []*v1.TextPart{{Text: "one ", Place: &v1.TextPlace{StartMs: 1500}}}, NextCursor: "1"}), nil
	}
	return connect.NewResponse(&v1.ReadTextResponse{Parts: []*v1.TextPart{{Text: "two"}}}), nil
}

func newTextClient(t *testing.T, h *textHost) *host.TextClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostTextServiceHandler(h))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return host.NewTextClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestTextClientSavesListsAndReadsThroughTheHost(t *testing.T) {
	ctx := t.Context()
	h := &textHost{}
	client := newTextClient(t, h)

	version, err := client.PutVersion(ctx, host.TextPut{Path: "a.pdf", Key: "k", Input: "q/x", Format: domain.TextFormatTimed, InputHash: "i", Hash: "h"})
	if err != nil {
		t.Fatal(err)
	}
	if h.request.GetFormat() != v1.TextFormat_TEXT_FORMAT_TIMED || h.request.GetInputVersionId() != "q/x" || h.auth != "Bearer tok" {
		t.Fatalf("the host received %v with %q", h.request, h.auth)
	}
	if version.GetID() != "p/k" || version.Format != domain.TextFormatTimed || !version.CreateTime.Equal(time.UnixMilli(1_700_000_000_000)) {
		t.Fatalf("version %+v", version)
	}

	listing, err := client.ListVersions(ctx, "a.pdf")
	if err != nil || listing.Live != "p/k" || len(listing.Versions) != 1 {
		t.Fatalf("listing %+v (%v)", listing, err)
	}
	text, err := host.ReadWholeText(ctx, client, listing.Versions[0])
	if err != nil || text.Text != "one two" || text.Locate(0) != "0:01" {
		t.Fatalf("read %q located %q (%v)", text.Text, text.Locate(0), err)
	}
}

func TestTheTextClientReadsEveryCodeAsItsError(t *testing.T) {
	for code, want := range map[v1.TextErrorCode]error{
		v1.TextErrorCode_TEXT_ERROR_CODE_MISSING:   host.ErrNotExist,
		v1.TextErrorCode_TEXT_ERROR_CODE_NO_VAULT:  host.ErrNoVault,
		v1.TextErrorCode_TEXT_ERROR_CODE_INVALID:   host.ErrInvalidVersion,
		v1.TextErrorCode_TEXT_ERROR_CODE_NOT_OWNER: host.ErrNotOwner,
	} {
		_, err := newTextClient(t, &textHost{code: code}).PutVersion(t.Context(), host.TextPut{Path: "a", Key: "k"})
		if !errors.Is(err, want) {
			t.Errorf("%s: got %v, want %v", code, err, want)
		}
	}
	if len(v1.TextErrorCode_name) != 5 {
		t.Fatalf("the schema names %d codes and this test reads 4 and the zero", len(v1.TextErrorCode_name))
	}
}

func TestAProviderIsServedAsTextProviderService(t *testing.T) {
	ctx := t.Context()
	provider := hosttest.NewMemoryProvider()
	provider.Hold("a.pdf", "k", []domain.TextPart{
		{Text: "one ", Section: "Intro", Place: domain.TextPlace{Page: 2, Rects: []domain.Rect{{MaxX: 1}}}},
		{Text: "two"},
	})
	mux := http.NewServeMux()
	mux.Handle(host.NewTextProviderHandler(provider))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewTextProviderServiceClient(server.Client(), server.URL)

	first, err := client.GetText(ctx, connect.NewRequest(&v1.TextProviderServiceGetTextRequest{VaultId: "v", Path: "a.pdf", Key: "k"}))
	if err != nil {
		t.Fatal(err)
	}
	parts := first.Msg.GetParts()
	if len(parts) != 1 || parts[0].GetSection() != "Intro" || parts[0].GetPlace().GetPage() != 2 || first.Msg.GetNextCursor() != "1" {
		t.Fatalf("first page %v", first.Msg)
	}
	missing, err := client.GetText(ctx, connect.NewRequest(&v1.TextProviderServiceGetTextRequest{Path: "a.pdf", Key: "gone"}))
	if err != nil || missing.Msg.GetError() != v1.TextErrorCode_TEXT_ERROR_CODE_MISSING {
		t.Fatalf("a text not there answered %v (%v)", missing.Msg, err)
	}
	if _, err := client.DeleteText(ctx, connect.NewRequest(&v1.TextProviderServiceDeleteTextRequest{Path: "a.pdf", Key: "k"})); err != nil {
		t.Fatal(err)
	}
	after, err := client.GetText(ctx, connect.NewRequest(&v1.TextProviderServiceGetTextRequest{Path: "a.pdf", Key: "k"}))
	if err != nil || after.Msg.GetError() != v1.TextErrorCode_TEXT_ERROR_CODE_MISSING {
		t.Fatalf("a deleted text answered %v (%v)", after.Msg, err)
	}
}

// recordingProvider is a provider that records each creation it is requests for.
type recordingProvider struct {
	hosttest.MemoryProvider
	requests []host.TextCreation
}

func (c *recordingProvider) CreateText(_ context.Context, req host.TextCreation) error {
	c.requests = append(c.requests, req)
	return nil
}

func TestProviderReceivesUserStartedFlag(t *testing.T) {
	provider := &recordingProvider{MemoryProvider: *hosttest.NewMemoryProvider()}
	mux := http.NewServeMux()
	mux.Handle(host.NewTextProviderHandler(provider))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewTextProviderServiceClient(server.Client(), server.URL)

	for _, isUserStarted := range []bool{true, false} {
		req := &v1.TextProviderServiceCreateTextRequest{VaultId: "v", Path: "a.pdf", InputVersionId: "p/k", IsUserStarted: isUserStarted}
		if _, err := client.CreateText(t.Context(), connect.NewRequest(req)); err != nil {
			t.Fatal(err)
		}
	}
	want := []host.TextCreation{
		{Vault: "v", Path: "a.pdf", Input: "p/k", IsUserStarted: true},
		{Vault: "v", Path: "a.pdf", Input: "p/k"},
	}
	if !slices.Equal(provider.requests, want) {
		t.Fatalf("got %+v, want %+v", provider.requests, want)
	}
}
