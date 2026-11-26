package service_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
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
	server := synctest.NewServer(t, clock)
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
	if len(vaults[1].RemoteID) != 32 {
		t.Fatalf("remote id = %q, want 32 hexadecimal digits", vaults[1].RemoteID)
	}
	want[1].RemoteID = vaults[1].RemoteID
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

func TestPreviewConnect_CountsTheNotesWithAnotherTextOnEachSide(t *testing.T) {
	_, backend, remoteID := newJoiningWorld(t)

	response, err := backend.GetConnectPreview(t.Context(), api.PreviewConnectRequest{VaultID: "Copy", RemoteID: remoteID})

	if err != nil {
		t.Fatal(err)
	}
	if response.DifferCount != 2 {
		t.Errorf("differ count = %d, want 2 (Plan and Ideas)", response.DifferCount)
	}
}

func TestPreviewConnect_DoesNotConnectTheVault(t *testing.T) {
	_, backend, remoteID := newJoiningWorld(t)

	if _, err := backend.GetConnectPreview(t.Context(), api.PreviewConnectRequest{VaultID: "Copy", RemoteID: remoteID}); err != nil {
		t.Fatal(err)
	}

	if vaults := listVaults(t, backend); len(vaults) != 1 || vaults[0].State != api.StateNotConnected {
		t.Errorf("vaults = %+v, want Copy not connected", vaults)
	}
}

func TestPreviewConnect_RefusesAnIdThatIsNotAnId(t *testing.T) {
	_, backend, _ := newJoiningWorld(t)

	_, err := backend.GetConnectPreview(t.Context(), api.PreviewConnectRequest{VaultID: "Copy", RemoteID: "not an id"})

	if err == nil {
		t.Error("PreviewConnect of a bad id succeeded, want an error")
	}
}

type meetingHost struct {
	ports.Host
	arrived atomic.Int32
}

func (h *meetingHost) GetVaults(ctx context.Context) ([]domain.Vault, error) {
	h.arrived.Add(1)
	for range 300 {
		if h.arrived.Load() >= 2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	return h.Host.GetVaults(ctx)
}

func TestConnectVault_ConnectsOnlyOneFolderToAVaultWhenTwoRequestsRace(t *testing.T) {
	w := newWorld(t)
	w.c.Connect(t, "Test")
	second := synctest.NewSecondComputer(t, w.server, w.clock, "Copy", w.words)
	second.AddVault(t, "Copy two")
	deps := second.SyncDeps()
	meeting := &meetingHost{Host: deps.Host}
	deps.Host = meeting
	backend := service.NewBackend(deps)
	remote, err := backend.ListRemoteVaults(t.Context(), api.ListRemoteVaultsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	meeting.arrived.Store(0)
	errs := make(chan error, 2)
	for _, vault := range []string{"Copy", "Copy two"} {
		go func() {
			errs <- connect(t, backend, api.ConnectVaultRequest{VaultID: vault, Mode: api.ModeExisting, RemoteID: remote.Vaults[0].RemoteID})
		}()
	}

	failed := 0
	for range 2 {
		if err := <-errs; err != nil {
			failed++
		}
	}

	if failed != 1 {
		t.Errorf("%d of 2 racing requests failed, want exactly 1", failed)
	}
}

func findVault(t *testing.T, vaults []api.Vault, id string) api.Vault {
	t.Helper()
	for _, one := range vaults {
		if one.ID == id {
			return one
		}
	}
	t.Fatalf("no vault %q in %+v", id, vaults)
	return api.Vault{}
}

func TestListVaults_ReportsAnIdleVaultWithEveryFileAtItsHeadAsNotSyncing(t *testing.T) {
	w := newWorld(t)
	w.c.Connect(t, "Test")
	w.c.WriteNote("Test", "Soup.md", "carrots")
	w.c.WriteNote("Test", "Bread.md", "flour and water")
	if err := w.c.Sync(t.Context(), "Test"); err != nil {
		t.Fatal(err)
	}
	backend := service.NewBackend(w.c.SyncDeps()).WithTransfers(w.c.State, w.c.State)

	one := findVault(t, listVaults(t, backend), "Test")

	if one.Syncing || one.PendingFiles != 0 {
		t.Errorf("vault = %+v, want it idle with nothing pending", one)
	}
	if one.SyncedBytes == 0 || one.SyncedBytes != one.TotalBytes {
		t.Errorf("synced = %d, total = %d, want both the same and above zero", one.SyncedBytes, one.TotalBytes)
	}
}

func TestListVaults_ReportsAbsentFilesAsOnDemandAndNotAsPendingWork(t *testing.T) {
	w := newWorld(t)
	w.c.Connect(t, "Test")
	w.c.WriteNote("Test", "Soup.md", "carrots")
	if err := w.c.Sync(t.Context(), "Test"); err != nil {
		t.Fatal(err)
	}
	absent := []host.AbsentFile{
		{Path: "Archive.pdf", Size: 5000, ModTime: time.Unix(0, 1), Hash: "00"},
		{Path: "Atlas.pdf", Size: 700, ModTime: time.Unix(0, 1), Hash: "01"},
	}
	if err := w.c.Host.GetAbsent("Test").PutAbsentFiles(t.Context(), absent); err != nil {
		t.Fatal(err)
	}
	backend := service.NewBackend(w.c.SyncDeps()).WithTransfers(w.c.State, w.c.State)

	one := findVault(t, listVaults(t, backend), "Test")

	if one.Syncing || one.PendingFiles != 0 {
		t.Errorf("vault = %+v, want it idle with nothing pending", one)
	}
	if one.OnDemandFiles != 2 || one.OnDemandBytes != 5700 {
		t.Errorf("on demand = %d files, %d bytes, want 2 files, 5700 bytes", one.OnDemandFiles, one.OnDemandBytes)
	}
	if one.SyncedBytes == 0 || one.TotalBytes != one.SyncedBytes+5700 {
		t.Errorf("synced = %d, total = %d, want total as synced plus the 5700 bytes in the cloud", one.SyncedBytes, one.TotalBytes)
	}
}

func TestListVaults_MarksVaultSyncingDuringPass(t *testing.T) {
	w := newWorld(t)
	w.c.Connect(t, "Test")
	backend := service.NewBackend(w.c.SyncDeps())
	unlock, err := w.c.State.LockVault(t.Context(), "Test")
	if err != nil {
		t.Fatal(err)
	}
	isSyncing := findVault(t, listVaults(t, backend), "Test").Syncing
	unlock()

	if !isSyncing || findVault(t, listVaults(t, backend), "Test").Syncing {
		t.Errorf("syncing while locked = %v, want true, and false once released", isSyncing)
	}
}
