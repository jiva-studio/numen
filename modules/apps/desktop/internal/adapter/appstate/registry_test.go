package appstate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

func newRegistry(t *testing.T) *appstate.VaultRegistry {
	t.Helper()
	return appstate.At(filepath.Join(t.TempDir(), "state", "vaults.json"))
}

func TestMissingFileIsAnEmptyList(t *testing.T) {
	// A first run has no registry yet, and that is not an error.
	got, err := newRegistry(t).All()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestSaveThenAll(t *testing.T) {
	r := newRegistry(t)
	v := domain.Vault{ID: "01AAA", Name: "personal", Path: "/home/user/notes"}
	if err := r.Save(v); err != nil {
		t.Fatal(err)
	}
	got, err := r.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != v {
		t.Errorf("got %v", got)
	}
}

func TestSavingTheSameIdentityMovesTheVault(t *testing.T) {
	r := newRegistry(t)
	if err := r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/old"}); err != nil {
		t.Fatal(err)
	}
	// The identity travels with the folder; the registry only remembers where it
	// was last seen, so re-adding must move the entry rather than duplicate it.
	if err := r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/new"}); err != nil {
		t.Fatal(err)
	}
	got, err := r.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1", len(got))
	}
	if got[0].Path != "/new" {
		t.Errorf("path = %s, want /new", got[0].Path)
	}
}

func TestFindAcceptsNamePathAndIdentity(t *testing.T) {
	r := newRegistry(t)
	dir := t.TempDir()
	v := domain.Vault{ID: "01AAA", Name: "Personal", Path: dir}
	if err := r.Save(v); err != nil {
		t.Fatal(err)
	}

	for _, typed := range []string{"01AAA", "Personal", "personal", dir} {
		got, found, err := r.Find(typed)
		if err != nil {
			t.Fatal(err)
		}
		if !found {
			t.Errorf("%q found nothing", typed)
			continue
		}
		if got.ID != v.ID {
			t.Errorf("%q found %s", typed, got.ID)
		}
	}

	if _, found, err := r.Find("nothing like this"); err != nil || found {
		t.Errorf("unknown name matched: %v %v", found, err)
	}
}

func TestFileIsReadableByAHuman(t *testing.T) {
	// The registry is JSON precisely so that it can be opened and fixed when the
	// application will not start, so its shape is part of the contract.
	path := filepath.Join(t.TempDir(), "vaults.json")
	r := appstate.At(path)
	if err := r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		V      int `json:"v"`
		Vaults []struct {
			ID, Name, Path string
		} `json:"vaults"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if parsed.V != 1 || len(parsed.Vaults) != 1 {
		t.Errorf("unexpected shape: %s", raw)
	}
}

func TestWriteDoesNotLeaveATemporaryFileBehind(t *testing.T) {
	dir := t.TempDir()
	r := appstate.At(filepath.Join(dir, "vaults.json"))
	if err := r.Save(domain.Vault{ID: "01AAA", Name: "x", Path: "/x"}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("directory holds %v, want only the registry", entries)
	}
}

func TestAFailedSaveLeavesTheOldRegistryIntact(t *testing.T) {
	// The registry is the one thing here whose loss costs the user manual work,
	// so a half-written file is the failure worth defending against. The write
	// goes through a temporary file and a rename; this makes the temporary write
	// fail and checks that what was already there survived.
	dir := t.TempDir()
	path := filepath.Join(dir, "vaults.json")
	r := appstate.At(path)

	original := domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}
	if err := r.Save(original); err != nil {
		t.Fatal(err)
	}

	// A directory where the temporary file wants to be: the write fails, the
	// rename never happens.
	if err := os.Mkdir(path+".tmp", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := r.Save(domain.Vault{ID: "01BBB", Name: "work", Path: "/work"}); err == nil {
		t.Fatal("a save that could not write reported success")
	}

	after, err := r.All()
	if err != nil {
		t.Fatalf("the registry is unreadable after a failed save: %v", err)
	}
	if len(after) != 1 || after[0] != original {
		t.Errorf("the previous registry did not survive: %+v", after)
	}
}
