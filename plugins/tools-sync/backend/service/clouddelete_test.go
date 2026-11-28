package service_test

import (
	"context"
	"errors"
	"testing"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/service"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	usecaseshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
)

type failingCloud struct {
	ports.CloudVaults
	err error
}

func (f failingCloud) DeleteVault(context.Context, []byte, []byte, []byte) (*syncv1.DeleteVaultResponse, error) {
	return nil, f.err
}

func syncedWorld(t *testing.T) (*world, *service.Backend, api.RemoteVault) {
	t.Helper()
	w := newWorld(t)
	w.c.Connect(t, "Test")
	w.c.WriteNote("Test", "Food/Soup.md", "carrots")
	if err := w.c.Sync(t.Context(), "Test"); err != nil {
		t.Fatal(err)
	}
	backend := service.NewBackend(w.c.SyncDeps())
	return w, backend, listRemote(t, backend)[0]
}

func listRemote(t *testing.T, backend *service.Backend) []api.RemoteVault {
	t.Helper()
	remote, err := backend.ListRemoteVaults(t.Context(), api.ListRemoteVaultsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	return remote.Vaults
}

func TestListRemoteVaults_ListsTheSizeInTheCloudAndAVaultThatStoppedSyncing(t *testing.T) {
	_, backend, remote := syncedWorld(t)
	if remote.SizeBytes == 0 {
		t.Errorf("size = %d, want the bytes of the synced note", remote.SizeBytes)
	}

	if _, err := backend.StopSyncingVault(t.Context(), api.StopSyncingVaultRequest{VaultID: "Test"}); err != nil {
		t.Fatal(err)
	}

	listed := listRemote(t, backend)
	if len(listed) != 1 || listed[0].IsConnected || listed[0].SizeBytes != remote.SizeBytes {
		t.Errorf("remote = %+v, want the vault kept, not connected, with the size %d", listed, remote.SizeBytes)
	}
}

func TestStopSyncingVault_DeletingTheCloudCopyRemovesTheVaultFromTheCloudAndKeepsTheFiles(t *testing.T) {
	w, backend, remote := syncedWorld(t)
	if err := w.c.CreateShare(t.Context(), "Test", "Food"); err != nil {
		t.Fatal(err)
	}

	_, err := backend.StopSyncingVault(t.Context(), api.StopSyncingVaultRequest{VaultID: "Test", DeleteCloudCopy: true})

	if err != nil {
		t.Fatal(err)
	}
	if got, ok := w.c.ReadNote("Test", "Food/Soup.md"); !ok || got != "carrots" {
		t.Errorf("note = %q (there: %v), want it kept", got, ok)
	}
	if vaults := listVaults(t, backend); vaults[0].State != api.StateNotConnected {
		t.Errorf("state = %q, want not connected", vaults[0].State)
	}
	if listed := listRemote(t, backend); len(listed) != 0 {
		t.Errorf("remote = %+v, want the vault gone", listed)
	}
	usage, err := w.c.Cloud.GetVaultUsage(t.Context())
	if err != nil || len(usage) != 0 {
		t.Errorf("server usage = %v, %v, want none: the vault %s is deleted", usage, err, remote.RemoteID)
	}
	if err := w.c.Sync(t.Context(), "Test"); !errors.Is(err, vaultlink.ErrNotConnected) {
		t.Errorf("sync error = %v, want the vault not connected", err)
	}
}

func TestStopSyncingVault_ClosesTheSharesAndKeepsTheVaultLinkedWhenTheServerRefusesTheDeletion(t *testing.T) {
	w, _, _ := syncedWorld(t)
	if err := w.c.CreateShare(t.Context(), "Test", "Food"); err != nil {
		t.Fatal(err)
	}
	deps := w.c.SyncDeps()
	deps.Cloud = failingCloud{CloudVaults: deps.Cloud, err: errors.New("the server is away")}
	backend := service.NewBackend(deps)

	_, err := backend.StopSyncingVault(t.Context(), api.StopSyncingVaultRequest{VaultID: "Test", DeleteCloudCopy: true})

	if err == nil {
		t.Fatal("the method did not fail")
	}
	if vaults := listVaults(t, backend); vaults[0].State != api.StateConnected {
		t.Errorf("state = %q, want connected", vaults[0].State)
	}
	roots, err := usecaseshare.NewReadShares(w.c.ShareDeps()).Execute(t.Context(), "Test")
	if err != nil || len(roots) != 1 || !roots[0].IsClosed {
		t.Errorf("shares = %+v, %v, want one, closed", roots, err)
	}
}

func TestStopSyncingVault_AnotherDeviceDropsTheLinkOfADeletedCloudCopy(t *testing.T) {
	w, backend, remote := syncedWorld(t)
	second := synctest.NewSecondComputer(t, w.server, w.clock, "Copy", w.words)
	if err := connect(t, service.NewBackend(second.SyncDeps()), api.ConnectVaultRequest{VaultID: "Copy", Mode: api.ModeExisting, RemoteID: remote.RemoteID}); err != nil {
		t.Fatal(err)
	}
	if err := second.Sync(t.Context(), "Copy"); err != nil {
		t.Fatal(err)
	}

	if _, err := backend.StopSyncingVault(t.Context(), api.StopSyncingVaultRequest{VaultID: "Test", DeleteCloudCopy: true}); err != nil {
		t.Fatal(err)
	}

	if err := second.Sync(t.Context(), "Copy"); !errors.Is(err, vaultlink.ErrNotConnected) {
		t.Errorf("sync error = %v, want the link dropped", err)
	}
	other := service.NewBackend(second.SyncDeps())
	if vaults := listVaults(t, other); vaults[0].State != api.StateNotConnected {
		t.Errorf("state = %q, want not connected", vaults[0].State)
	}
	if got, ok := second.ReadNote("Copy", "Food/Soup.md"); !ok || got != "carrots" {
		t.Errorf("note = %q (there: %v), want it kept", got, ok)
	}
	if listed := listRemote(t, other); len(listed) != 0 {
		t.Errorf("remote = %+v, want the vault gone", listed)
	}
}

func TestDeleteCloudVault_DeletesAVaultThatIsNotSyncedOnThisDevice(t *testing.T) {
	w, backend, remote := syncedWorld(t)
	if _, err := backend.StopSyncingVault(t.Context(), api.StopSyncingVaultRequest{VaultID: "Test"}); err != nil {
		t.Fatal(err)
	}

	_, err := backend.DeleteCloudVault(t.Context(), api.DeleteCloudVaultRequest{RemoteID: remote.RemoteID})

	if err != nil {
		t.Fatal(err)
	}
	if listed := listRemote(t, backend); len(listed) != 0 {
		t.Errorf("remote = %+v, want the vault gone", listed)
	}
	if usage, err := w.c.Cloud.GetVaultUsage(t.Context()); err != nil || len(usage) != 0 {
		t.Errorf("server usage = %v, %v, want none", usage, err)
	}
	if _, err := backend.DeleteCloudVault(t.Context(), api.DeleteCloudVaultRequest{RemoteID: remote.RemoteID}); err != nil {
		t.Errorf("a second deletion failed: %v", err)
	}
	if _, err := backend.DeleteCloudVault(t.Context(), api.DeleteCloudVaultRequest{RemoteID: "zz"}); err == nil {
		t.Error("an id that is not hexadecimal was accepted")
	}
}
