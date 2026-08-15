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
	got, err := newRegistry(t).List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestAddThenList(t *testing.T) {
	r := newRegistry(t)
	v := domain.Vault{ID: "01AAA", Name: "personal", Path: "/home/user/notes"}
	if err := r.Add(v); err != nil {
		t.Fatal(err)
	}
	got, err := r.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != v {
		t.Errorf("got %v", got)
	}
}

func TestAddingTheSameIdentityMovesTheVault(t *testing.T) {
	r := newRegistry(t)
	if err := r.Add(domain.Vault{ID: "01AAA", Name: "personal", Path: "/old"}); err != nil {
		t.Fatal(err)
	}
	// The identity travels with the folder; the registry only remembers where it
	// was last seen, so re-adding must move the entry rather than duplicate it.
	if err := r.Add(domain.Vault{ID: "01AAA", Name: "personal", Path: "/new"}); err != nil {
		t.Fatal(err)
	}
	got, err := r.List()
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
	if err := r.Add(v); err != nil {
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
	if err := r.Add(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err != nil {
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
	if err := r.Add(domain.Vault{ID: "01AAA", Name: "x", Path: "/x"}); err != nil {
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
