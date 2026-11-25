package hoststore_test

import (
	"context"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hoststore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

const vaultName = "first"

type countingHost struct {
	ports.Host
	writes *atomic.Int64
}

func (h countingHost) Store(id domain.VaultID) ports.Store {
	return countingStore{Store: h.Host.Store(id), writes: h.writes}
}

type countingStore struct {
	ports.Store
	writes *atomic.Int64
}

func (s countingStore) Write(ctx context.Context, name string, content []byte) error {
	s.writes.Add(1)
	return s.Store.Write(ctx, name, content)
}

func newCounted(t *testing.T) (*hoststore.State, *atomic.Int64, domain.VaultID) {
	t.Helper()
	vault := hosttest.CreateOnDisk(t, vaultName, nil)
	writes := &atomic.Int64{}
	return hoststore.New(countingHost{Host: synctest.Host{Root: vault}, writes: writes}), writes, vault.Vault().ID
}

func fileID(n int) []byte { return binary.BigEndian.AppendUint32(nil, uint32(n)) }

func TestUpdateLink_TwoWritersLoseNoUpdate(t *testing.T) {
	state, _, id := newCounted(t)
	if err := state.WriteLink(t.Context(), id, vaultlink.Link{RemoteVaultID: "00"}); err != nil {
		t.Fatal(err)
	}
	const writers = 40
	var group sync.WaitGroup
	for n := range writers {
		group.Go(func() {
			err := state.UpdateLink(t.Context(), id, func(link *vaultlink.Link) error {
				if n%2 == 0 {
					link.AddRemoved(vaultlink.RemovedFile{FileID: fileID(n), Version: uint64(n + 1)})
				} else {
					link.AddRetired(vaultlink.RetiredFile{FileID: fileID(n), Version: uint64(n + 1)})
				}
				return nil
			})
			if err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()

	link, err := state.ReadLink(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(link.Removed) != writers/2 || len(link.Retired) != writers/2 {
		t.Errorf("removed %d, retired %d, want %d each", len(link.Removed), len(link.Retired), writers/2)
	}
}

func TestUpdateLink_ANotConnectedVaultIsRefused(t *testing.T) {
	state, _, id := newCounted(t)

	err := state.UpdateLink(t.Context(), id, func(*vaultlink.Link) error { return nil })

	if err != vaultlink.ErrNotConnected {
		t.Errorf("err = %v, want ErrNotConnected", err)
	}
}

func TestWriteLink_AnUnchangedLinkIsNotWrittenAgain(t *testing.T) {
	state, writes, id := newCounted(t)
	link := vaultlink.Link{RemoteVaultID: "00", ServerSeq: 4}
	if err := state.WriteLink(t.Context(), id, link); err != nil {
		t.Fatal(err)
	}
	read, err := state.ReadLink(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	before := writes.Load()

	if err := state.WriteLink(t.Context(), id, read); err != nil {
		t.Fatal(err)
	}
	read.ServerSeq = 5
	if err := state.WriteLink(t.Context(), id, read); err != nil {
		t.Fatal(err)
	}

	if got := writes.Load() - before; got != 1 {
		t.Errorf("writes = %d, want 1: only the changed link", got)
	}
}

func TestLockVault_OnePassHoldsTheVaultAtATime(t *testing.T) {
	state, _, id := newCounted(t)
	var inside, overlaps atomic.Int64
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			unlock, err := state.LockVault(t.Context(), id)
			if err != nil {
				t.Error(err)
				return
			}
			defer unlock()
			if inside.Add(1) > 1 {
				overlaps.Add(1)
			}
			inside.Add(-1)
		})
	}
	group.Wait()

	if overlaps.Load() != 0 {
		t.Errorf("%d passes ran inside another", overlaps.Load())
	}
}

func TestLockVault_AWaitingPassStopsWithItsContext(t *testing.T) {
	state, _, id := newCounted(t)
	unlock, err := state.LockVault(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := state.LockVault(ctx, id); err == nil {
		t.Error("a cancelled wait got the vault")
	}
}
