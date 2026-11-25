package hoststore_test

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hoststore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

func newState(vaults ...*hosttest.Vault) *hoststore.State {
	for _, other := range vaults[1:] {
		vaults[0].LinkBeside(other)
	}
	return hoststore.New(synctest.Host{Root: vaults[0]})
}

func TestAVaultWithNoJournalFileHasAnEmptyJournal(t *testing.T) {
	state := newState(hosttest.CreateOnDisk(t, "first", nil))

	journal, err := state.ReadJournal(t.Context(), restore.KindCatalog)

	if err != nil || len(journal.Entries) != 0 || journal.Kind != restore.KindCatalog {
		t.Fatalf("got %+v, %v", journal, err)
	}
}

func TestWriteJournal_SkipsMissingFolders(t *testing.T) {
	first := hosttest.CreateOnDisk(t, "first", nil)
	gone := hosttest.CreateOnDisk(t, "gone", nil)
	if err := os.RemoveAll(gone.Vault().Path); err != nil {
		t.Fatal(err)
	}
	state := newState(first, gone)
	want := restore.Journal{Kind: restore.KindCatalog, Entries: [][]byte{[]byte("a")}}

	if err := state.WriteJournal(t.Context(), want); err != nil {
		t.Fatalf("a vault whose folder is gone failed the write: %v", err)
	}

	got, err := state.ReadJournal(t.Context(), restore.KindCatalog)
	if err != nil || len(got.Entries) != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestWritePin_SkipsMissingFolders(t *testing.T) {
	first := hosttest.CreateOnDisk(t, "first", nil)
	gone := hosttest.CreateOnDisk(t, "gone", nil)
	if err := os.RemoveAll(gone.Vault().Path); err != nil {
		t.Fatal(err)
	}
	state := newState(first, gone)

	if err := state.WritePin(t.Context(), "sync.example", []byte("key")); err != nil {
		t.Fatalf("a vault whose folder is gone failed the write: %v", err)
	}

	got, err := state.ReadPin(t.Context(), "sync.example")
	if err != nil || string(got) != "key" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestWriteJournal_FailsWhenNoFolderExists(t *testing.T) {
	only := hosttest.CreateOnDisk(t, "only", nil)
	if err := os.RemoveAll(only.Vault().Path); err != nil {
		t.Fatal(err)
	}

	err := newState(only).WriteJournal(t.Context(), restore.Journal{Kind: restore.KindCatalog, Entries: nil})

	if err == nil {
		t.Fatal("a journal kept nowhere must be an error")
	}
}

func TestWritePin_ConcurrentWrites(t *testing.T) {
	state := newState(hosttest.CreateOnDisk(t, "first", nil))
	const count = 32
	var group sync.WaitGroup
	errs := make(chan error, count)
	for i := range count {
		group.Go(func() {
			errs <- state.WritePin(t.Context(), fmt.Sprintf("sync-%d.example", i), []byte{byte(i + 1)})
		})
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := range count {
		if key, err := state.ReadPin(t.Context(), fmt.Sprintf("sync-%d.example", i)); err != nil || !bytes.Equal(key, []byte{byte(i + 1)}) {
			t.Errorf("pin %d = %v (%v), want it kept", i, key, err)
		}
	}
}

func TestReadEntrySightings_AreNilBeforeAnyWrite(t *testing.T) {
	state := newState(hosttest.CreateOnDisk(t, "first", nil))

	got, err := state.ReadEntrySightings(t.Context())

	if err != nil || got != nil {
		t.Fatalf("got %v, %v, want nil and no error", got, err)
	}
}

func TestEntrySightings_ReturnWhatWasWritten(t *testing.T) {
	state := newState(hosttest.CreateOnDisk(t, "first", nil), hosttest.CreateOnDisk(t, "second", nil))
	want := map[string]int64{"a": 200, "b": 50}
	if err := state.WriteEntrySightings(t.Context(), want); err != nil {
		t.Fatal(err)
	}

	got, err := state.ReadEntrySightings(t.Context())

	if err != nil || len(got) != 2 || got["a"] != 200 || got["b"] != 50 {
		t.Errorf("got %v, %v, want %v", got, err, want)
	}
}
