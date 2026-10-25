package hostendpoint

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/vaultaccess"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

func (m *memoryVault) OpenStore(pluginID string) (pluginport.PluginStore, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.vault == nil {
		return nil, vaultaccess.ErrNoVault
	}
	if m.storeRoot == "" {
		return nil, errors.New("the vault keeps no store")
	}
	return derivedfile.PluginStores{}.Open(domain.Vault{ID: m.vault.ID, Path: m.storeRoot}, pluginID)
}

// storeHost is the store endpoint served over one vault whose store is a folder of the test's own.
type storeHost struct {
	vault  *memoryVault
	url    string
	client *http.Client
}

func newStoreHost(t *testing.T) storeHost {
	t.Helper()
	vault := newMemoryVault(nil)
	vault.storeRoot = t.TempDir()
	hosted := newCallers(nil)
	hosted.vault = vault
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostStoreServiceHandler(&storeEndpoint{endpoint: &Endpoint{callers: hosted}}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return storeHost{vault: vault, url: server.URL, client: server.Client()}
}

// newClient is a client calling the store as the plugin the token names.
func (h storeHost) newClient(token string) numenv1connect.HostStoreServiceClient {
	return numenv1connect.NewHostStoreServiceClient(h.client, h.url, connect.WithInterceptors(transport.AuthInterceptor(token)))
}

func TestAPluginsFilesAreAppendedReadAndListedInItsOwnFolder(t *testing.T) {
	h := newStoreHost(t)
	client := h.newClient("token-a")
	ctx := t.Context()

	for _, line := range []string{"one\n", "two\n"} {
		if _, err := client.AppendStoreFile(ctx, connect.NewRequest(&v1.AppendStoreFileRequest{Name: "a.log", Content: []byte(line)})); err != nil {
			t.Fatal(err)
		}
	}
	read, readErr := client.ReadStoreFile(ctx, connect.NewRequest(&v1.ReadStoreFileRequest{Name: "a.log"}))
	listed, listErr := client.ListStoreFiles(ctx, connect.NewRequest(&v1.ListStoreFilesRequest{}))
	on, diskErr := os.ReadFile(filepath.Join(h.vault.storeRoot, ".numen", "md.numen.a", "a.log"))
	missing, missingErr := client.ReadStoreFile(ctx, connect.NewRequest(&v1.ReadStoreFileRequest{Name: "none"}))

	if readErr != nil || string(read.Msg.GetContent()) != "one\ntwo\n" {
		t.Errorf("read %v (%v)", read.Msg, readErr)
	}
	if listErr != nil || len(listed.Msg.GetFiles()) != 1 || listed.Msg.GetFiles()[0].GetName() != "a.log" || listed.Msg.GetFiles()[0].GetSize() != 8 {
		t.Errorf("listed %v (%v)", listed.Msg, listErr)
	}
	if diskErr != nil || string(on) != "one\ntwo\n" {
		t.Errorf("the folder holds %q (%v)", on, diskErr)
	}
	if missingErr != nil || missing.Msg.GetError() != v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_MISSING {
		t.Errorf("a missing file was answered %v (%v)", missing.Msg, missingErr)
	}
}

func TestAPluginSeesNoOtherPluginsFolder(t *testing.T) {
	h := newStoreHost(t)
	ctx := t.Context()
	if _, err := h.newClient("token-a").AppendStoreFile(ctx, connect.NewRequest(&v1.AppendStoreFileRequest{Name: "secret", Content: []byte("a")})); err != nil {
		t.Fatal(err)
	}
	other := h.newClient("token-b")

	mine, err := other.ReadStoreFile(ctx, connect.NewRequest(&v1.ReadStoreFileRequest{Name: "secret"}))
	if err != nil || mine.Msg.GetError() != v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_MISSING {
		t.Errorf("another plugin's file was answered %v (%v)", mine.Msg, err)
	}
	listed, err := other.ListStoreFiles(ctx, connect.NewRequest(&v1.ListStoreFilesRequest{}))
	if err != nil || len(listed.Msg.GetFiles()) != 0 {
		t.Errorf("another plugin listed %v (%v)", listed.Msg, err)
	}
	for _, name := range []string{"../md.numen.a/secret", "../config.json", ".hidden", ""} {
		res, err := other.ReadStoreFile(ctx, connect.NewRequest(&v1.ReadStoreFileRequest{Name: name}))
		if err != nil || res.Msg.GetError() != v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_OUTSIDE {
			t.Errorf("%q was answered %v (%v)", name, res.Msg, err)
		}
	}
}

func TestThePluginsFolderTakesNoVaultPermissionButATokenTheHostGave(t *testing.T) {
	h := newStoreHost(t)
	ctx := t.Context()

	_, err := h.newClient("token-b").AppendStoreFile(ctx, connect.NewRequest(&v1.AppendStoreFileRequest{Name: "a", Content: []byte("a")}))
	if err != nil {
		t.Errorf("a plugin without vault permissions was refused its folder: %v", err)
	}
	_, err = h.newClient("token-x").ReadStoreFile(ctx, connect.NewRequest(&v1.ReadStoreFileRequest{Name: "a"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("an unknown token was answered %v", err)
	}
}

func TestAHostShowingNoVaultAnswersNoVaultForTheStore(t *testing.T) {
	h := newStoreHost(t)
	h.vault.mu.Lock()
	h.vault.vault = nil
	h.vault.mu.Unlock()

	res, err := h.newClient("token-a").ReadStoreFile(t.Context(), connect.NewRequest(&v1.ReadStoreFileRequest{Name: "a"}))

	if err != nil || res.Msg.GetError() != v1.HostStoreErrorCode_HOST_STORE_ERROR_CODE_NO_VAULT {
		t.Errorf("answered %v (%v)", res.Msg, err)
	}
}
