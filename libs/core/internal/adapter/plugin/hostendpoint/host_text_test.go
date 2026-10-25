package hostendpoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// memoryTexts keeps the versions the plugins put, and answers a version's text with its key.
type memoryTexts struct {
	mu       sync.Mutex
	versions []domain.TextVersion
}

func (m *memoryTexts) PutVersion(_ context.Context, pluginID string, params pluginport.PluginTextPut) (domain.TextVersion, error) {
	if params.Key == "" || !params.Format.IsValid() {
		return domain.TextVersion{}, textport.ErrInvalidVersion
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	version := domain.TextVersion{
		Path: params.Path, PluginID: pluginID, Key: params.Key, Input: params.Input, Format: params.Format,
		InputHash: params.InputHash, Hash: params.Hash,
	}
	m.versions = append(m.versions, version)
	return version, nil
}

func (m *memoryTexts) RemoveVersion(_ context.Context, pluginID, path, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.versions = slices.DeleteFunc(m.versions, func(v domain.TextVersion) bool {
		return v.Path == path && v.GetID() == domain.NewTextVersionID(pluginID, key)
	})
	return nil
}

func (m *memoryTexts) ListVersions(_ context.Context, path string) (pluginport.PluginTextListing, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var listing pluginport.PluginTextListing
	for _, v := range m.versions {
		if v.Path == path {
			listing.Versions = append(listing.Versions, v)
			listing.Live = v.GetID()
		}
	}
	return listing, nil
}

func (m *memoryTexts) ReadText(_ context.Context, path string, id domain.TextVersionID, _ string) (pluginport.PluginTextPage, error) {
	listing, _ := m.ListVersions(context.Background(), path)
	for _, v := range listing.Versions {
		if v.GetID() == id {
			return pluginport.PluginTextPage{Parts: []domain.TextPart{{Text: v.Key, Place: domain.TextPlace{Page: 3}}}}, nil
		}
	}
	return pluginport.PluginTextPage{}, textport.ErrNoText
}

func newTextClient(t *testing.T, fakeCallers *callers, token string) numenv1connect.HostTextServiceClient {
	t.Helper()
	endpoint := &Endpoint{callers: fakeCallers}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostTextServiceHandler(&textEndpoint{endpoint: endpoint}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewHostTextServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor(token)))
}

func TestHostEndpointSavesListsReadsAndRemovesVersions(t *testing.T) {
	ctx := t.Context()
	fakeCallers := newCallers(nil)
	fakeCallers.texts = &memoryTexts{}
	client := newTextClient(t, fakeCallers, "token-a")

	put, err := client.PutTextVersion(ctx, connect.NewRequest(&v1.PutTextVersionRequest{
		Path: "a.pdf", Key: "k", Format: v1.TextFormat_TEXT_FORMAT_POSITIONED, InputHash: "f", Hash: "h",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if put.Msg.GetError() != v1.TextErrorCode_TEXT_ERROR_CODE_UNSPECIFIED || put.Msg.GetVersion().GetId() != "md.numen.a/k" ||
		put.Msg.GetVersion().GetPluginId() != "md.numen.a" {
		t.Fatalf("put answered %v, want the version recorded as the calling plugin's", put.Msg)
	}

	listed, err := client.ListTextVersions(ctx, connect.NewRequest(&v1.ListTextVersionsRequest{Path: "a.pdf"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Msg.GetVersions()) != 1 || listed.Msg.GetLiveVersionId() != "md.numen.a/k" {
		t.Fatalf("listed %v", listed.Msg)
	}

	read, err := client.ReadText(ctx, connect.NewRequest(&v1.ReadTextRequest{Path: "a.pdf", VersionId: "md.numen.a/k"}))
	if err != nil {
		t.Fatal(err)
	}
	if parts := read.Msg.GetParts(); len(parts) != 1 || parts[0].GetText() != "k" || parts[0].GetPlace().GetPage() != 3 {
		t.Fatalf("read %v", read.Msg)
	}
	missing, err := client.ReadText(ctx, connect.NewRequest(&v1.ReadTextRequest{Path: "a.pdf", VersionId: "x/y"}))
	if err != nil {
		t.Fatal(err)
	}
	if missing.Msg.GetError() != v1.TextErrorCode_TEXT_ERROR_CODE_MISSING {
		t.Fatalf("a version that is not there answered %v", missing.Msg)
	}

	if _, err := client.RemoveTextVersion(ctx, connect.NewRequest(&v1.RemoveTextVersionRequest{Path: "a.pdf", Key: "k"})); err != nil {
		t.Fatal(err)
	}
	listed, err = client.ListTextVersions(ctx, connect.NewRequest(&v1.ListTextVersionsRequest{Path: "a.pdf"}))
	if err != nil || len(listed.Msg.GetVersions()) != 0 {
		t.Fatalf("after removing: %v (%v)", listed.Msg, err)
	}
	if !slices.Contains(fakeCallers.checked, "md.numen.a vault:write") || !slices.Contains(fakeCallers.checked, "md.numen.a vault:read") {
		t.Fatalf("checked %v, want both permissions asked", fakeCallers.checked)
	}
}

func TestAnInvalidVersionIsAnsweredWithItsCode(t *testing.T) {
	fakeCallers := newCallers(nil)
	fakeCallers.texts = &memoryTexts{}
	res, err := newTextClient(t, fakeCallers, "token-a").PutTextVersion(t.Context(), connect.NewRequest(&v1.PutTextVersionRequest{Path: "a.pdf", Key: "k"}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != v1.TextErrorCode_TEXT_ERROR_CODE_INVALID || res.Msg.GetVersion() != nil {
		t.Fatalf("answered %v, want INVALID and no version", res.Msg)
	}
}

func TestTextCallsWithNoVaultOrNoPermissionAreAnswered(t *testing.T) {
	fakeCallers := newCallers(nil)
	res, err := newTextClient(t, fakeCallers, "token-a").ListTextVersions(t.Context(), connect.NewRequest(&v1.ListTextVersionsRequest{Path: "a.pdf"}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != v1.TextErrorCode_TEXT_ERROR_CODE_NO_VAULT {
		t.Fatalf("a host showing no vault answered %v", res.Msg)
	}
	fakeCallers.texts = &memoryTexts{}
	_, err = newTextClient(t, fakeCallers, "token-b").PutTextVersion(t.Context(), connect.NewRequest(&v1.PutTextVersionRequest{
		Path: "a.pdf", Key: "k", Format: v1.TextFormat_TEXT_FORMAT_PLAIN,
	}))
	if code := readCode(t, err); code != connect.CodePermissionDenied {
		t.Fatalf("a plugin without the permission got %v", code)
	}
	_, err = newTextClient(t, fakeCallers, "token-unknown").ListTextVersions(t.Context(), connect.NewRequest(&v1.ListTextVersionsRequest{}))
	if code := readCode(t, err); code != connect.CodeUnauthenticated {
		t.Fatalf("an unknown token got %v", code)
	}
}
