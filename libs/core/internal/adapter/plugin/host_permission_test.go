package plugin

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifestjson"
)

// createManifestedHost returns a host that treats each plugin as running with the
// permissions its manifest declares. The manifest is decoded by the production decoder.
func createManifestedHost(t *testing.T, manifests map[string]string) *Host {
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

// createRawVaultClient returns a generated protocol client of the host endpoint that carries the token only:
// no plugin SDK check runs before the call reaches the host.
func createRawVaultClient(t *testing.T, host *Host, token string) numenv1connect.HostVaultServiceClient {
	t.Helper()
	endpoint, err := hostendpoint.Open(t.Context(), host.hostAddress, host)
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
