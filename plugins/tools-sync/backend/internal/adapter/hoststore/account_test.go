package hoststore_test

import (
	"os"
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

func TestTheJournalIsKeptInTheVaultsWhoseFolderIsThere(t *testing.T) {
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

func TestAPinIsKeptInTheVaultsWhoseFolderIsThere(t *testing.T) {
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

func TestNoJournalIsKeptWhenEveryFolderIsGone(t *testing.T) {
	only := hosttest.CreateOnDisk(t, "only", nil)
	if err := os.RemoveAll(only.Vault().Path); err != nil {
		t.Fatal(err)
	}

	err := newState(only).WriteJournal(t.Context(), restore.Journal{Kind: restore.KindCatalog, Entries: nil})

	if err == nil {
		t.Fatal("a journal kept nowhere must be an error")
	}
}
