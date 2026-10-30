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

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// textHost answers HostTextService with what a test set, and records what it was put and the vault
// each call named.
type textHost struct {
	numenv1connect.UnimplementedHostTextServiceHandler
	code    v1.TextErrorCode
	request *v1.PutTextVersionRequest
	auth    string
	vaults  []string
}

func (h *textHost) PutTextVersion(_ context.Context, req *connect.Request[v1.PutTextVersionRequest]) (*connect.Response[v1.PutTextVersionResponse], error) {
	h.request, h.auth = req.Msg, req.Header().Get("Authorization")
	h.vaults = append(h.vaults, req.Header().Get("Numen-Vault"))
	if h.code != v1.TextErrorCode_TEXT_ERROR_CODE_UNSPECIFIED {
		return connect.NewResponse(&v1.PutTextVersionResponse{Error: h.code}), nil
	}
	return connect.NewResponse(&v1.PutTextVersionResponse{Version: &v1.TextVersion{
		Id: "p/" + req.Msg.GetKey(), Path: req.Msg.GetPath(), PluginId: "p", Key: req.Msg.GetKey(),
		Kind: req.Msg.GetKind(), File: req.Msg.GetFile(), CreateTimeUnixMs: 1_700_000_000_000,
	}}), nil
}

func (h *textHost) ListTextVersions(_ context.Context, req *connect.Request[v1.ListTextVersionsRequest]) (*connect.Response[v1.ListTextVersionsResponse], error) {
	h.vaults = append(h.vaults, req.Header().Get("Numen-Vault"))
	return connect.NewResponse(&v1.ListTextVersionsResponse{
		Versions:      []*v1.TextVersion{{Id: "p/k", Path: "a.pdf", PluginId: "p", Key: "k", Kind: "transcript"}},
		LiveVersionId: "p/k",
		Error:         h.code,
	}), nil
}

func (h *textHost) ReadText(_ context.Context, req *connect.Request[v1.ReadTextRequest]) (*connect.Response[v1.ReadTextResponse], error) {
	h.vaults = append(h.vaults, req.Header().Get("Numen-Vault"))
	if req.Msg.GetCursor() == "" {
		return connect.NewResponse(&v1.ReadTextResponse{Segments: []*v1.TextSegment{{Text: "one ", Selector: `{"t":[1.5,2]}`}}, NextCursor: "1"}), nil
	}
	return connect.NewResponse(&v1.ReadTextResponse{Segments: []*v1.TextSegment{{Text: "two"}}}), nil
}

func (h *textHost) RemoveTextVersion(_ context.Context, req *connect.Request[v1.RemoveTextVersionRequest]) (*connect.Response[v1.RemoveTextVersionResponse], error) {
	h.vaults = append(h.vaults, req.Header().Get("Numen-Vault"))
	return connect.NewResponse(&v1.RemoveTextVersionResponse{}), nil
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

	version, err := client.PutVersion(ctx, host.TextPut{Path: "a.pdf", Key: "k", Source: "q/x", Kind: "transcript", SourceHash: "i", File: "k.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	if h.request.GetKind() != "transcript" || h.request.GetSourceVersionId() != "q/x" || h.request.GetFile() != "k.jsonl" || h.auth != "Bearer tok" {
		t.Fatalf("the host received %v with %q", h.request, h.auth)
	}
	if version.GetID() != "p/k" || version.Kind != "transcript" || version.File != "k.jsonl" || !version.CreateTime.Equal(time.UnixMilli(1_700_000_000_000)) {
		t.Fatalf("version %+v", version)
	}

	listing, err := client.ListVersions(ctx, "a.pdf")
	if err != nil || listing.Live != "p/k" || len(listing.Versions) != 1 {
		t.Fatalf("listing %+v (%v)", listing, err)
	}
	text, err := host.ReadWholeText(ctx, client, listing.Versions[0])
	if err != nil || text.Text != "one two" || text.GetSelector(0) != `{"t":[1.5,2]}` {
		t.Fatalf("read %q (%v)", text.Text, err)
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

// recordingProvider is a provider that records each creation it is requests for.
type recordingProvider struct {
	requests []host.TextCreation
}

func (c *recordingProvider) CreateText(_ context.Context, req host.TextCreation) error {
	c.requests = append(c.requests, req)
	return nil
}

func TestProviderReceivesUserStartedFlag(t *testing.T) {
	provider := &recordingProvider{}
	mux := http.NewServeMux()
	mux.Handle(host.NewTextProviderHandler(provider))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := numenv1connect.NewTextProviderServiceClient(server.Client(), server.URL)

	for _, isUserStarted := range []bool{true, false} {
		req := &v1.TextProviderServiceCreateTextRequest{VaultId: "v", Path: "a.pdf", SourceVersionId: "p/k", IsUserStarted: isUserStarted}
		if _, err := client.CreateText(t.Context(), connect.NewRequest(req)); err != nil {
			t.Fatal(err)
		}
	}
	want := []host.TextCreation{
		{Vault: "v", Path: "a.pdf", Source: "p/k", IsUserStarted: true},
		{Vault: "v", Path: "a.pdf", Source: "p/k"},
	}
	if !slices.Equal(provider.requests, want) {
		t.Fatalf("got %+v, want %+v", provider.requests, want)
	}
}

func TestATextClientOfAnotherVaultNamesItOnEveryCall(t *testing.T) {
	ctx := t.Context()
	h := &textHost{}
	client := newTextClient(t, h)
	chemistry := client.With("Chemistry")

	if _, err := chemistry.PutVersion(ctx, host.TextPut{Path: "a.pdf", Key: "k", Kind: "transcript", File: "k.jsonl"}); err != nil {
		t.Fatal(err)
	}
	if _, err := chemistry.ListVersions(ctx, "a.pdf"); err != nil {
		t.Fatal(err)
	}
	if _, err := chemistry.ReadText(ctx, "a.pdf", "p/k", ""); err != nil {
		t.Fatal(err)
	}
	if err := chemistry.RemoveVersion(ctx, "a.pdf", "k"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListVersions(ctx, "a.pdf"); err != nil {
		t.Fatal(err)
	}

	want := []string{"Chemistry", "Chemistry", "Chemistry", "Chemistry", ""}
	if !slices.Equal(h.vaults, want) {
		t.Errorf("the vaults named were %q, want %q", h.vaults, want)
	}
}

func TestATextClientOfAnUnknownVaultReturnsErrUnknownVault(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostTextServiceHandler(unknownVaultTexts{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := host.NewTextClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"}).With("Unknown")

	if _, err := client.ListVersions(t.Context(), "a.pdf"); !errors.Is(err, host.ErrUnknownVault) {
		t.Errorf("an unknown vault returned %v", err)
	}
}

// unknownVaultTexts answers every call as a host that knows no vault of the name.
type unknownVaultTexts struct {
	numenv1connect.UnimplementedHostTextServiceHandler
}

func (unknownVaultTexts) ListTextVersions(context.Context, *connect.Request[v1.ListTextVersionsRequest]) (*connect.Response[v1.ListTextVersionsResponse], error) {
	return nil, connect.NewError(connect.CodeNotFound, errors.New("no such vault"))
}
