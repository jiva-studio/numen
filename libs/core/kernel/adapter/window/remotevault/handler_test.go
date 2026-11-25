package remotevault_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/remotevault"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfolder"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type provider struct {
	listed       []vaultdomain.RemoteVault
	listFailure  error
	connectError error
}

func (p *provider) ListRemoteVaults(context.Context) ([]vaultdomain.RemoteVault, error) {
	return p.listed, p.listFailure
}

func (p *provider) ConnectRemoteVault(context.Context, string, domain.VaultID) error {
	return p.connectError
}

type userFolders struct{ home, documents string }

func (u userFolders) GetHomeFolder() (string, error)      { return u.home, nil }
func (u userFolders) GetDocumentsFolder() (string, error) { return u.documents, nil }

type fixture struct {
	client   numenv1connect.RemoteVaultServiceClient
	provider *provider
	registry *appstate.VaultRegistry
	parent   string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	registry := appstate.OpenAt(filepath.Join(t.TempDir(), "state", "vaults.json"))
	clock := testsupport.GetMachine()
	parent := t.TempDir()
	user := userFolders{home: filepath.Join(parent, "home"), documents: filepath.Join(parent, "Documents")}
	remote := &provider{}
	add := vaults.NewAdd(vaultreader.Identity{}, registry, registry, registry, clock)
	handler := remotevault.NewHandler(
		vaults.NewListRemoteVaults(remote),
		vaults.NewSuggestVaultPath(func() string { return "" }, user, vaultfolder.Folders{}),
		vaults.NewOpenRemoteVault(registry, registry, vaultfolder.Folders{}, user, add, remote, clock),
	)
	mux := http.NewServeMux()
	mux.Handle(remotevault.NewRoute(handler, connect.WithInterceptors()))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &fixture{
		client:   numenv1connect.NewRemoteVaultServiceClient(server.Client(), server.URL),
		provider: remote, registry: registry, parent: parent,
	}
}

func getCode(err error) connect.Code {
	var failure *connect.Error
	if !errors.As(err, &failure) {
		return connect.CodeUnknown
	}
	return failure.Code()
}

func TestTheRemoteVaultsOfTheProviderAreListed(t *testing.T) {
	f := newFixture(t)
	f.provider.listed = []vaultdomain.RemoteVault{{RemoteID: "r-1", Name: "Notes", SizeBytes: 9, VaultID: "v-1"}, {RemoteID: "r-2", Name: "Books"}}

	got, err := f.client.ListRemoteVaults(t.Context(), connect.NewRequest(&v1.RemoteVaultServiceListRemoteVaultsRequest{}))

	if err != nil {
		t.Fatal(err)
	}
	listed := got.Msg.GetVaults()
	if len(listed) != 2 || listed[0].GetRemoteId() != "r-1" || listed[0].GetName() != "Notes" || listed[0].GetSizeBytes() != 9 ||
		listed[0].GetVaultId() != "v-1" || listed[1].GetVaultId() != "" {
		t.Errorf("the list is %v", listed)
	}
}

func TestAProviderThatIsNotAvailableIsAnEmptyListAndNoError(t *testing.T) {
	f := newFixture(t)
	f.provider.listFailure = &plugindomain.UnavailableError{Capability: "vault.provider", Reason: plugindomain.NotInstalled}

	got, err := f.client.ListRemoteVaults(t.Context(), connect.NewRequest(&v1.RemoteVaultServiceListRemoteVaultsRequest{}))

	if err != nil {
		t.Fatalf("the call failed: %v", err)
	}
	if len(got.Msg.GetVaults()) != 0 {
		t.Errorf("the list is %v, want none", got.Msg.GetVaults())
	}
}

func TestANewVaultIsSuggestedAFolderInTheDocumentsFolder(t *testing.T) {
	f := newFixture(t)

	got, err := f.client.SuggestVaultPath(t.Context(), connect.NewRequest(&v1.SuggestVaultPathRequest{Name: "Notes"}))

	if err != nil {
		t.Fatal(err)
	}
	documents := filepath.Join(f.parent, "Documents")
	if got.Msg.GetFolder() != documents || got.Msg.GetPath() != filepath.Join(documents, "Notes") || got.Msg.GetIsTaken() {
		t.Errorf("the suggestion is %v", got.Msg)
	}
}

func TestARemoteVaultIsOpenedAndTheVaultIdIsAnswered(t *testing.T) {
	f := newFixture(t)
	path := filepath.Join(f.parent, "Notes")

	got, err := f.client.OpenRemoteVault(t.Context(), connect.NewRequest(&v1.OpenRemoteVaultRequest{RemoteId: "r-1", Path: path}))

	if err != nil {
		t.Fatal(err)
	}
	listed, listErr := f.registry.List()
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(listed) != 1 || string(listed[0].ID) != got.Msg.GetVaultId() {
		t.Errorf("the list holds %+v, the answer is %q", listed, got.Msg.GetVaultId())
	}
}

func TestAPathThatCannotBeAVaultIsAnInvalidArgumentAndMakesNothing(t *testing.T) {
	f := newFixture(t)

	_, err := f.client.OpenRemoteVault(t.Context(), connect.NewRequest(&v1.OpenRemoteVaultRequest{RemoteId: "r-1", Path: "Notes"}))

	if getCode(err) != connect.CodeInvalidArgument {
		t.Errorf("the call failed with %v, want an invalid argument", err)
	}
	if _, statErr := os.Stat("Notes"); !os.IsNotExist(statErr) {
		t.Errorf("a folder was made: %v", statErr)
	}
}

func TestAMissingRemoteVaultIsAnInvalidArgument(t *testing.T) {
	f := newFixture(t)

	_, err := f.client.OpenRemoteVault(t.Context(), connect.NewRequest(&v1.OpenRemoteVaultRequest{Path: filepath.Join(f.parent, "Notes")}))

	if getCode(err) != connect.CodeInvalidArgument {
		t.Errorf("the call failed with %v, want an invalid argument", err)
	}
}

func TestOpenRemoteVault_ProviderFailure_ReturnsItsCodeAndMessage(t *testing.T) {
	f := newFixture(t)
	f.provider.connectError = connect.NewError(connect.CodeFailedPrecondition, errors.New("Sign in to open the vault"))
	path := filepath.Join(f.parent, "Notes")

	_, err := f.client.OpenRemoteVault(t.Context(), connect.NewRequest(&v1.OpenRemoteVaultRequest{RemoteId: "r-1", Path: path}))

	var failure *connect.Error
	if !errors.As(err, &failure) || failure.Code() != connect.CodeFailedPrecondition || failure.Message() != "Sign in to open the vault" {
		t.Errorf("the call failed with %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("the folder of the failed call is still there: %v", statErr)
	}
}

func TestAProviderThatIsNotAvailableIsAFailedPreconditionOnOpening(t *testing.T) {
	f := newFixture(t)
	f.provider.connectError = &plugindomain.UnavailableError{Capability: "vault.provider", Reason: plugindomain.NotInstalled}

	_, err := f.client.OpenRemoteVault(t.Context(), connect.NewRequest(&v1.OpenRemoteVaultRequest{RemoteId: "r-1", Path: filepath.Join(f.parent, "Notes")}))

	if getCode(err) != connect.CodeFailedPrecondition {
		t.Errorf("the call failed with %v, want a failed precondition", err)
	}
}

func TestANewVaultIsSuggestedAFolderInTheChosenFolder(t *testing.T) {
	f := newFixture(t)
	chosen := filepath.Join(f.parent, "Music")
	if err := os.MkdirAll(filepath.Join(chosen, "Notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := f.client.SuggestVaultPath(t.Context(), connect.NewRequest(&v1.SuggestVaultPathRequest{Name: "Notes", Folder: chosen}))

	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetFolder() != chosen || got.Msg.GetPath() != filepath.Join(chosen, "Notes 2") || !got.Msg.GetIsTaken() {
		t.Errorf("the suggestion is %v", got.Msg)
	}
}
