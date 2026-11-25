package unlink_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hoststore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/unlink"
)

func TestStopSyncing_WaitsForTheHolderOfTheVaultBeforeForgettingIt(t *testing.T) {
	vault := hosttest.CreateOnDisk(t, "first", nil)
	state := hoststore.New(synctest.Host{Root: vault})
	id := vault.Vault().ID
	if err := state.WriteLink(t.Context(), id, vaultlink.Link{RemoteVaultID: "00"}); err != nil {
		t.Fatal(err)
	}
	unlock, err := state.LockVault(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	stop := unlink.NewStopSyncing(unlink.Deps{Links: state, Forget: state, Vaults: nil, Cloud: nil, Locker: state})
	done := make(chan error, 1)
	go func() { done <- stop.Execute(t.Context(), id, false) }()

	select {
	case err := <-done:
		t.Fatalf("stopped while another pass holds the vault: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if _, err := state.ReadLink(t.Context(), id); err != nil {
		t.Fatalf("link forgotten under a holder: %v", err)
	}
	unlock()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
