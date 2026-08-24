package appstate_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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

// TestNothingNamesNoVault. A path is resolved against the folder this process
// was started in, and a vault standing there is still not what nothing names.
func TestNothingNamesNoVault(t *testing.T) {
	here, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	r := newRegistry(t)
	if err := r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: here}); err != nil {
		t.Fatal(err)
	}

	got, found, err := r.Find("")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Errorf("nothing named the vault %s at %s", got.Name, got.Path)
	}
}

func TestRemoveTakesAVaultOffTheList(t *testing.T) {
	r := newRegistry(t)
	kept := domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}
	if err := r.Save(kept); err != nil {
		t.Fatal(err)
	}
	if err := r.Save(domain.Vault{ID: "01BBB", Name: "work", Path: "/work"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Remove("01BBB"); err != nil {
		t.Fatal(err)
	}
	got, err := r.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != kept {
		t.Errorf("got %v, want only %v", got, kept)
	}
}

func TestRemovingAVaultThatIsNotOnTheListIsNotAnError(t *testing.T) {
	r := newRegistry(t)
	if err := r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Remove("01NEVERADDED"); err != nil {
		t.Errorf("removing what is not there: %v", err)
	}
	got, err := r.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("got %v, want the list untouched", got)
	}
}

func TestTheVaultOpenedLastSurvivesARoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vaults.json")
	v := domain.Vault{ID: "01BBB", Name: "work", Path: "/work"}
	first := appstate.At(path)
	if err := first.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err != nil {
		t.Fatal(err)
	}
	if err := first.Save(v); err != nil {
		t.Fatal(err)
	}
	if err := first.Opened(v.ID); err != nil {
		t.Fatal(err)
	}

	// A second registry over the same file is the next run of the application.
	got, found, err := appstate.At(path).Last()
	if err != nil {
		t.Fatal(err)
	}
	if !found || got != v {
		t.Errorf("last = %v %v, want %v", got, found, v)
	}
}

func TestNothingWasOpenedUntilAVaultIs(t *testing.T) {
	r := newRegistry(t)
	if err := r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err != nil {
		t.Fatal(err)
	}
	if got, found, err := r.Last(); err != nil || found {
		t.Errorf("last = %v %v %v, want nothing", got, found, err)
	}
}

func TestRemovingTheVaultOpenedLastForgetsThatItWas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vaults.json")
	r := appstate.At(path)
	for _, v := range []domain.Vault{
		{ID: "01AAA", Name: "personal", Path: "/notes"},
		{ID: "01BBB", Name: "work", Path: "/work"},
	} {
		if err := r.Save(v); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Opened("01BBB"); err != nil {
		t.Fatal(err)
	}
	if err := r.Remove("01BBB"); err != nil {
		t.Fatal(err)
	}

	if got, found, err := r.Last(); err != nil || found {
		t.Errorf("last = %v %v %v, want nothing", got, found, err)
	}
	// The file no longer names it either: an identity nothing on the list
	// carries is not what a person reading this file should find.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "01BBB") {
		t.Errorf("the removed vault is still named in the file: %s", raw)
	}
}

func TestOpeningAVaultThatIsNotOnTheListIsRefused(t *testing.T) {
	r := newRegistry(t)
	if err := r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Opened("01NEVERADDED"); err == nil {
		t.Fatal("an identity nothing on the list carries was recorded")
	}
	if got, found, err := r.Last(); err != nil || found {
		t.Errorf("last = %v %v %v, want nothing", got, found, err)
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

func TestTheFileNamesTheVaultOpenedLast(t *testing.T) {
	// Part of the same contract: a person reading the file sees which vault the
	// application will open, and the key is absent until one has been opened.
	path := filepath.Join(t.TempDir(), "vaults.json")
	r := appstate.At(path)
	if err := r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err != nil {
		t.Fatal(err)
	}

	shape := func() map[string]any {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("not valid JSON: %v", err)
		}
		return parsed
	}

	if _, present := shape()["last"]; present {
		t.Errorf("nothing has been opened, and the file says one was: %v", shape())
	}
	if err := r.Opened("01AAA"); err != nil {
		t.Fatal(err)
	}
	if got := shape()["last"]; got != "01AAA" {
		t.Errorf("last = %v, want 01AAA", got)
	}
}

func TestConcurrentWritesLoseNothing(t *testing.T) {
	// Every write rewrites the whole file, so two of them starting from the same
	// list is one list saved. The palette, the agent's tools and the command
	// line all write here.
	const each = 10
	r := newRegistry(t)
	for i := range each {
		if err := r.Save(domain.Vault{ID: fmt.Sprintf("01OLD%02d", i), Name: fmt.Sprintf("old %d", i), Path: fmt.Sprintf("/old/%d", i)}); err != nil {
			t.Fatal(err)
		}
	}

	var writing sync.WaitGroup
	errs := make(chan error, 2*each)
	for i := range each {
		writing.Add(2)
		go func() {
			defer writing.Done()
			errs <- r.Save(domain.Vault{ID: fmt.Sprintf("01NEW%02d", i), Name: fmt.Sprintf("new %d", i), Path: fmt.Sprintf("/new/%d", i)})
		}()
		go func() {
			defer writing.Done()
			errs <- r.Remove(fmt.Sprintf("01OLD%02d", i))
		}()
	}
	writing.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}

	got, err := r.All()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, v := range got {
		names = append(names, v.ID)
	}
	slices.Sort(names)
	var want []string
	for i := range each {
		want = append(want, fmt.Sprintf("01NEW%02d", i))
	}
	if !slices.Equal(names, want) {
		t.Errorf("the list holds %v, want %v", names, want)
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
