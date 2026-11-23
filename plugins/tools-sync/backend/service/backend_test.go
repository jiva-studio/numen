package service_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/service"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

type world struct {
	server *synctest.Server
	clock  *synctest.Clock
	words  []string
	c      *synctest.Computer
}

func newWorld(t *testing.T) *world {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	return &world{server: server, clock: clock, words: words, c: synctest.NewFirstComputer(t, server, clock, "Test", words)}
}

func listVaults(t *testing.T, backend *service.Backend) []api.Vault {
	t.Helper()
	response, err := backend.ListVaults(t.Context(), api.ListVaultsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	return response.Vaults
}

func connect(t *testing.T, backend *service.Backend, request api.ConnectVaultRequest) error {
	t.Helper()
	_, err := backend.ConnectVault(t.Context(), request)
	return err
}

func TestListVaults_ListsEveryVaultOfTheInstallationWithItsState(t *testing.T) {
	w := newWorld(t)
	w.c.AddVault(t, "Plain")
	w.c.AddVault(t, "Test copy")
	w.c.Connect(t, "Test")
	w.c.Vaults.Store().With("Test copy").WriteFile(synctest.LinkFile, w.c.ReadStoreFile(t, "Test", synctest.LinkFile))
	backend := service.NewBackend(w.c.SyncDeps())

	vaults := listVaults(t, backend)

	want := []api.Vault{
		{ID: "Plain", Name: "Plain", Path: "/vaults/Plain", State: api.StateNotConnected},
		{ID: "Test", Name: "Test", Path: "/vaults/Test", State: api.StateConnected},
		{ID: "Test copy", Name: "Test copy", Path: "/vaults/Test copy", State: api.StateCopy},
	}
	if len(vaults) != len(want) {
		t.Fatalf("vaults = %+v, want %d", vaults, len(want))
	}
	for i := range want {
		if vaults[i] != want[i] {
			t.Errorf("vault %d = %+v, want %+v", i, vaults[i], want[i])
		}
	}
}

func TestConnectVault_ConnectsAVaultAsANewSyncedVault(t *testing.T) {
	w := newWorld(t)
	backend := service.NewBackend(w.c.SyncDeps())

	if err := connect(t, backend, api.ConnectVaultRequest{VaultID: "Test", Mode: api.ModeNew}); err != nil {
		t.Fatal(err)
	}

	if vaults := listVaults(t, backend); len(vaults) != 1 || vaults[0].State != api.StateConnected {
		t.Errorf("vaults = %+v, want Test connected", vaults)
	}
	remote, err := backend.ListRemoteVaults(t.Context(), api.ListRemoteVaultsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(remote.Vaults) != 1 || remote.Vaults[0].Name != "Test" || !remote.Vaults[0].IsConnected || len(remote.Vaults[0].RemoteID) != 32 {
		t.Errorf("remote vaults = %+v, want Test, connected, with an id of 32 hexadecimal digits", remote.Vaults)
	}
}

func TestConnectVault_ConnectsAFolderOfASecondComputerToAnExistingVault(t *testing.T) {
	w := newWorld(t)
	w.c.Connect(t, "Test")
	second := synctest.NewSecondComputer(t, w.server, w.clock, "Copy", w.words)
	backend := service.NewBackend(second.SyncDeps())
	remote, err := backend.ListRemoteVaults(t.Context(), api.ListRemoteVaultsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(remote.Vaults) != 1 || remote.Vaults[0].IsConnected {
		t.Fatalf("remote vaults = %+v, want Test, not connected", remote.Vaults)
	}

	err = connect(t, backend, api.ConnectVaultRequest{VaultID: "Copy", Mode: api.ModeExisting, RemoteID: remote.Vaults[0].RemoteID})

	if err != nil {
		t.Fatal(err)
	}
	if vaults := listVaults(t, backend); len(vaults) != 1 || vaults[0].State != api.StateConnected {
		t.Errorf("vaults = %+v, want Copy connected", vaults)
	}
}

func TestConnectVault_WarnsAboutACopyOfAConnectedVault(t *testing.T) {
	w := newWorld(t)
	w.c.AddVault(t, "Test copy")
	w.c.Connect(t, "Test")
	w.c.Vaults.Store().With("Test copy").WriteFile(synctest.LinkFile, w.c.ReadStoreFile(t, "Test", synctest.LinkFile))
	backend := service.NewBackend(w.c.SyncDeps())

	err := connect(t, backend, api.ConnectVaultRequest{VaultID: "Test copy", Mode: api.ModeNew})

	if err == nil || !strings.Contains(err.Error(), "a copy of a connected vault") {
		t.Errorf("error = %v, want the text a copy of a connected vault", err)
	}
	if got := len(w.c.ReadCatalog(t)); got != 1 {
		t.Errorf("catalog has %d entries, want 1", got)
	}
}

func TestConnectVault_RefusesAModeItDoesNotKnow(t *testing.T) {
	w := newWorld(t)
	backend := service.NewBackend(w.c.SyncDeps())

	err := connect(t, backend, api.ConnectVaultRequest{VaultID: "Test", Mode: "both"})

	if err == nil {
		t.Error("ConnectVault of an unknown mode succeeded, want an error")
	}
	if got := len(w.c.ReadCatalog(t)); got != 0 {
		t.Errorf("catalog has %d entries, want none", got)
	}
}

func TestGetStatus_ReportsAnIdentityAndAReachableServer(t *testing.T) {
	w := newWorld(t)
	backend := service.NewBackend(w.c.SyncDeps())

	status, err := backend.GetStatus(t.Context(), api.GetStatusRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if !status.HasIdentity || !status.IsReachable || len(status.Items) != 0 {
		t.Errorf("status = %+v, want an identity, a reachable server and no item", status)
	}
}

func TestGetStatus_WarnsThatThisDeviceHasNoIdentity(t *testing.T) {
	w := newWorld(t)
	w.c.Fake.RemoveIdentity()
	backend := service.NewBackend(w.c.SyncDeps())

	status, err := backend.GetStatus(t.Context(), api.GetStatusRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if status.HasIdentity || len(status.Items) != 1 || status.Items[0].Text != "No identity on this device" {
		t.Errorf("status = %+v, want no identity and the item No identity on this device", status)
	}
}

func TestGetStatus_ReportsAServerThatIsAway(t *testing.T) {
	w := newWorld(t)
	w.server.SetDown(true)
	backend := service.NewBackend(w.c.SyncDeps())

	status, err := backend.GetStatus(t.Context(), api.GetStatusRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if !status.HasIdentity || status.IsReachable {
		t.Errorf("status = %+v, want an identity and an unreachable server", status)
	}
}

func newJoiningWorld(t *testing.T) (*world, *service.Backend, string) {
	t.Helper()
	w := newWorld(t)
	w.c.Connect(t, "Test")
	for path, text := range map[string]string{"Shopping list.md": "milk", "Plan.md": "tea", "Ideas.md": "walk"} {
		w.c.WriteNote("Test", path, text)
	}
	if err := w.c.Sync(t.Context(), "Test"); err != nil {
		t.Fatal(err)
	}
	second := synctest.NewSecondComputer(t, w.server, w.clock, "Copy", w.words)
	backend := service.NewBackend(second.SyncDeps())
	remote, err := backend.ListRemoteVaults(t.Context(), api.ListRemoteVaultsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	second.WriteNote("Copy", "Shopping list.md", "milk")
	second.WriteNote("Copy", "Plan.md", "eggs")
	second.WriteNote("Copy", "Ideas.md", "run")
	return w, backend, remote.Vaults[0].RemoteID
}

func TestGetConnectPreview_CountsTheNotesWithAnotherTextOnEachSide(t *testing.T) {
	_, backend, remoteID := newJoiningWorld(t)

	response, err := backend.GetConnectPreview(t.Context(), api.GetConnectPreviewRequest{VaultID: "Copy", RemoteID: remoteID})

	if err != nil {
		t.Fatal(err)
	}
	if response.DifferCount != 2 {
		t.Errorf("differ count = %d, want 2 (Plan and Ideas)", response.DifferCount)
	}
}

func TestGetConnectPreview_DoesNotConnectTheVault(t *testing.T) {
	_, backend, remoteID := newJoiningWorld(t)

	if _, err := backend.GetConnectPreview(t.Context(), api.GetConnectPreviewRequest{VaultID: "Copy", RemoteID: remoteID}); err != nil {
		t.Fatal(err)
	}

	if vaults := listVaults(t, backend); len(vaults) != 1 || vaults[0].State != api.StateNotConnected {
		t.Errorf("vaults = %+v, want Copy not connected", vaults)
	}
}

func TestGetConnectPreview_RefusesAnIdThatIsNotAnId(t *testing.T) {
	_, backend, _ := newJoiningWorld(t)

	_, err := backend.GetConnectPreview(t.Context(), api.GetConnectPreviewRequest{VaultID: "Copy", RemoteID: "not an id"})

	if err == nil {
		t.Error("GetConnectPreview of a bad id succeeded, want an error")
	}
}
