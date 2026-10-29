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

// memoryTexts keeps the versions the plugins put, by vault, and returns a version's text as its key.
type memoryTexts struct {
	mu       sync.Mutex
	versions map[domain.VaultID][]domain.TextVersion
}

func newMemoryTexts() *memoryTexts {
	return &memoryTexts{versions: map[domain.VaultID][]domain.TextVersion{}}
}

func (m *memoryTexts) PutVersion(_ context.Context, v domain.Vault, pluginID string, params pluginport.PluginTextPut) (domain.TextVersion, error) {
	if params.Key == "" || params.Kind == "" || params.File == "" {
		return domain.TextVersion{}, textport.ErrInvalidVersion
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	version := domain.TextVersion{
		Path: params.Path, PluginID: pluginID, Key: params.Key, Source: params.Source, Kind: params.Kind,
		SourceHash: params.SourceHash, File: params.File,
	}
	m.versions[v.ID] = append(m.versions[v.ID], version)
	return version, nil
}

func (m *memoryTexts) RemoveVersion(_ context.Context, v domain.Vault, pluginID, path, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.versions[v.ID] = slices.DeleteFunc(m.versions[v.ID], func(one domain.TextVersion) bool {
		return one.Path == path && one.GetID() == domain.NewTextVersionID(pluginID, key)
	})
	return nil
}

func (m *memoryTexts) ListVersions(_ context.Context, v domain.Vault, path string) (pluginport.PluginTextListing, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var listing pluginport.PluginTextListing
	for _, one := range m.versions[v.ID] {
		if one.Path == path {
			listing.Versions = append(listing.Versions, one)
			listing.Live = one.GetID()
		}
	}
	return listing, nil
}

func (m *memoryTexts) ReadText(_ context.Context, v domain.Vault, path string, id domain.TextVersionID, _ string) (pluginport.PluginTextPage, error) {
	listing, _ := m.ListVersions(context.Background(), v, path)
	for _, one := range listing.Versions {
		if one.GetID() == id {
			return pluginport.PluginTextPage{Segments: []domain.TextSegment{{Text: one.Key, Selector: `{"page":3}`}}}, nil
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
	fakeCallers.vault = newMemoryVault(nil)
	fakeCallers.texts = newMemoryTexts()
	client := newTextClient(t, fakeCallers, "token-a")

	put, err := client.PutTextVersion(ctx, connect.NewRequest(&v1.PutTextVersionRequest{
		Path: "a.pdf", Key: "k", Kind: "recognition", SourceHash: "f", File: "k.jsonl",
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
	if segments := read.Msg.GetSegments(); len(segments) != 1 || segments[0].GetText() != "k" || segments[0].GetSelector() != `{"page":3}` {
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

func TestHostText_InvalidVersionReturnsErrorCode(t *testing.T) {
	fakeCallers := newCallers(nil)
	fakeCallers.vault = newMemoryVault(nil)
	fakeCallers.texts = newMemoryTexts()
	res, err := newTextClient(t, fakeCallers, "token-a").PutTextVersion(t.Context(), connect.NewRequest(&v1.PutTextVersionRequest{Path: "a.pdf", Key: "k"}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != v1.TextErrorCode_TEXT_ERROR_CODE_INVALID || res.Msg.GetVersion() != nil {
		t.Fatalf("answered %v, want INVALID and no version", res.Msg)
	}
}

func TestHostText_ErrorWithoutVaultOrPermission(t *testing.T) {
	fakeCallers := newCallers(nil)
	res, err := newTextClient(t, fakeCallers, "token-a").ListTextVersions(t.Context(), connect.NewRequest(&v1.ListTextVersionsRequest{Path: "a.pdf"}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != v1.TextErrorCode_TEXT_ERROR_CODE_NO_VAULT {
		t.Fatalf("a host showing no vault answered %v", res.Msg)
	}
	fakeCallers.texts = newMemoryTexts()
	_, err = newTextClient(t, fakeCallers, "token-b").PutTextVersion(t.Context(), connect.NewRequest(&v1.PutTextVersionRequest{
		Path: "a.pdf", Key: "k", Kind: "article", File: "k.jsonl",
	}))
	if code := readCode(t, err); code != connect.CodePermissionDenied {
		t.Fatalf("a plugin without the permission got %v", code)
	}
	_, err = newTextClient(t, fakeCallers, "token-unknown").ListTextVersions(t.Context(), connect.NewRequest(&v1.ListTextVersionsRequest{}))
	if code := readCode(t, err); code != connect.CodeUnauthenticated {
		t.Fatalf("an unknown token got %v", code)
	}
}

func TestTextCallsWithoutVaultReturnNoVault(t *testing.T) {
	fakeCallers := newCallers(nil)
	fakeCallers.texts = newMemoryTexts()
	res, err := newTextClient(t, fakeCallers, "token-a").ListTextVersions(t.Context(), connect.NewRequest(&v1.ListTextVersionsRequest{Path: "a.pdf"}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != v1.TextErrorCode_TEXT_ERROR_CODE_NO_VAULT {
		t.Fatalf("a host with texts and no vault answered %v", res.Msg)
	}
}

func TestTextCallsUseVaultFromHeader(t *testing.T) {
	ctx := t.Context()
	physics := newMemoryVault(nil)
	physics.others = map[domain.VaultID]*memoryVault{
		"v2": {vault: &domain.Vault{ID: "v2", Name: "Chemistry", Path: "/vaults/Chemistry"}},
	}
	fakeCallers := newCallers(nil)
	fakeCallers.vault = physics
	texts := newMemoryTexts()
	fakeCallers.texts = texts
	client := newTextClient(t, fakeCallers, "token-a")

	putRequest := connect.NewRequest(&v1.PutTextVersionRequest{Path: "a.pdf", Key: "k", Kind: "recognition", SourceHash: "f", File: "k.jsonl"})
	putRequest.Header().Set("Numen-Vault", "v2")
	if _, err := client.PutTextVersion(ctx, putRequest); err != nil {
		t.Fatal(err)
	}
	list := connect.NewRequest(&v1.ListTextVersionsRequest{Path: "a.pdf"})
	list.Header().Set("Numen-Vault", "v2")
	listed, err := client.ListTextVersions(ctx, list)
	if err != nil {
		t.Fatal(err)
	}
	unknown := connect.NewRequest(&v1.ListTextVersionsRequest{Path: "a.pdf"})
	unknown.Header().Set("Numen-Vault", "v9")
	_, rejected := client.ListTextVersions(ctx, unknown)

	if len(texts.versions["v2"]) != 1 || len(texts.versions["v1"]) != 0 {
		t.Fatalf("versions by vault %v, want the version in v2 and none in the shown v1", texts.versions)
	}
	if len(listed.Msg.GetVersions()) != 1 {
		t.Fatalf("v2 listed %v", listed.Msg)
	}
	if connect.CodeOf(rejected) != connect.CodeNotFound {
		t.Fatalf("an unknown vault was answered %v", rejected)
	}
}
