package plugin_test

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func createManifestedHost(t *testing.T, manifests map[string]string) *container.PluginRoles {
	t.Helper()
	plugins := map[string]runningPlugin{}
	for token, text := range manifests {
		manifest, err := manifestjson.Read(strings.NewReader(text))
		if err != nil {
			t.Fatal(err)
		}
		plugins[manifest.ID] = runningPlugin{token: token, perms: manifest.Permissions}
	}
	return newRunningTestHost(t, plugins)
}

func createRawVaultClient(t *testing.T, host *container.PluginRoles, token string) numenv1connect.HostVaultServiceClient {
	t.Helper()
	endpoint, err := hostendpoint.Open(t.Context(), hostendpoint.Address, lentCallers(host))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := endpoint.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	return numenv1connect.NewHostVaultServiceClient(
		&http.Client{Timeout: 5 * time.Second},
		"http://127.0.0.1:"+strconv.Itoa(endpoint.GetPort()),
		connect.WithInterceptors(transport.AuthInterceptor(token)),
	)
}

// The core refuses a call whose permission the manifest of the calling plugin does not declare,
// whatever client the plugin frontend or backend calls with.
func TestACallWithoutTheManifestPermissionIsRefusedByTheCore(t *testing.T) {
	host := createManifestedHost(t, map[string]string{
		"token-read":  `{"id":"md.numen.reader","name":"Reader","version":"1.0.0","protocol":1,"permissions":["vault:read"]}`,
		"token-write": `{"id":"md.numen.writer","name":"Writer","version":"1.0.0","protocol":1,"permissions":["vault:write"]}`,
		"token-none":  `{"id":"md.numen.none","name":"None","version":"1.0.0","protocol":1,"permissions":[]}`,
	})
	for _, one := range []struct {
		token  string
		isRead bool
		denied bool
	}{
		{"token-none", true, true},
		{"token-none", false, true},
		{"token-write", true, true},
		{"token-read", false, true},
		{"token-read", true, false},
		{"token-write", false, false},
	} {
		client := createRawVaultClient(t, host, one.token)
		var err error
		if one.isRead {
			_, err = client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "a.md"}))
		} else {
			_, err = client.CreateFile(t.Context(), connect.NewRequest(&v1.CreateFileRequest{Path: "b.md"}))
		}
		got := connect.CodeOf(err) == connect.CodePermissionDenied
		if got != one.denied {
			t.Errorf("%s (read %v): permission denied is %v (%v), want %v", one.token, one.isRead, got, err, one.denied)
		}
	}
}

func createRawSearchClient(t *testing.T, host *container.PluginRoles, token string) numenv1connect.HostSearchServiceClient {
	t.Helper()
	endpoint, err := hostendpoint.Open(t.Context(), hostendpoint.Address, lentCallers(host))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := endpoint.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	return numenv1connect.NewHostSearchServiceClient(
		&http.Client{Timeout: 5 * time.Second},
		"http://127.0.0.1:"+strconv.Itoa(endpoint.GetPort()),
		connect.WithInterceptors(transport.AuthInterceptor(token)),
	)
}

// The index reads need index:read and the floor needs search:policy, whatever client the plugin calls with.
func TestASearchCallWithoutTheManifestPermissionIsRefusedByTheCore(t *testing.T) {
	host := createManifestedHost(t, map[string]string{
		"token-index":  `{"id":"md.numen.indexer","name":"Indexer","version":"1.0.0","protocol":1,"permissions":["index:read"]}`,
		"token-policy": `{"id":"md.numen.policy","name":"Policy","version":"1.0.0","protocol":1,"permissions":["search:policy"]}`,
		"token-vault":  `{"id":"md.numen.vault","name":"Vault","version":"1.0.0","protocol":1,"permissions":["vault:read","vault:write"]}`,
	})
	for _, one := range []struct {
		token string
		name  string
		call  func(ctx context.Context, client numenv1connect.HostSearchServiceClient) error
		deny  bool
	}{
		{"token-vault", "state", callGetEmbeddingState, true},
		{"token-vault", "stats", callGetIndexStats, true},
		{"token-vault", "sample", callSampleSimilarities, true},
		{"token-vault", "get floor", callGetSearchFloor, true},
		{"token-vault", "set floor", callSetSearchFloor, true},
		{"token-index", "state", callGetEmbeddingState, false},
		{"token-index", "stats", callGetIndexStats, false},
		{"token-index", "sample", callSampleSimilarities, false},
		{"token-index", "get floor", callGetSearchFloor, true},
		{"token-index", "set floor", callSetSearchFloor, true},
		{"token-policy", "state", callGetEmbeddingState, true},
		{"token-policy", "stats", callGetIndexStats, true},
		{"token-policy", "sample", callSampleSimilarities, true},
		{"token-policy", "get floor", callGetSearchFloor, false},
		{"token-policy", "set floor", callSetSearchFloor, false},
	} {
		err := one.call(t.Context(), createRawSearchClient(t, host, one.token))
		if got := connect.CodeOf(err) == connect.CodePermissionDenied; got != one.deny {
			t.Errorf("%s %s: permission denied is %v (%v), want %v", one.token, one.name, got, err, one.deny)
		}
	}
}

func callGetEmbeddingState(ctx context.Context, client numenv1connect.HostSearchServiceClient) error {
	_, err := client.GetEmbeddingState(ctx, connect.NewRequest(&v1.GetEmbeddingStateRequest{}))
	return err
}

func callSampleSimilarities(ctx context.Context, client numenv1connect.HostSearchServiceClient) error {
	_, err := client.SampleSimilarities(ctx, connect.NewRequest(&v1.SampleSimilaritiesRequest{Recipe: "r"}))
	return err
}

func callGetSearchFloor(ctx context.Context, client numenv1connect.HostSearchServiceClient) error {
	_, err := client.GetSearchFloor(ctx, connect.NewRequest(&v1.GetSearchFloorRequest{Recipe: "r"}))
	return err
}

func callSetSearchFloor(ctx context.Context, client numenv1connect.HostSearchServiceClient) error {
	_, err := client.SetSearchFloor(ctx, connect.NewRequest(&v1.SetSearchFloorRequest{Recipe: "r", Floor: 0.5}))
	return err
}

// A call with a token no running plugin was issued is unauthenticated, not served.
func TestACallWithAnUnknownTokenIsRefusedByTheCore(t *testing.T) {
	host := createManifestedHost(t, map[string]string{
		"token-read": `{"id":"md.numen.reader","name":"Reader","version":"1.0.0","protocol":1,"permissions":["vault:read"]}`,
	})
	client := createRawVaultClient(t, host, "token-forged")
	_, err := client.ReadFile(t.Context(), connect.NewRequest(&v1.ReadFileRequest{Path: "a.md"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("a forged token gave %v", err)
	}
}

func callGetIndexStats(ctx context.Context, client numenv1connect.HostSearchServiceClient) error {
	_, err := client.GetIndexStats(ctx, connect.NewRequest(&v1.GetIndexStatsRequest{}))
	return err
}
