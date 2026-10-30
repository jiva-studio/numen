package vault_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/text/unicode/norm"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// newAdd is the use case over a registry of its own, and the registry, which a
// test reads to see what was written.
func newAdd(t *testing.T) (vaults.Add, *appstate.VaultRegistry) {
	t.Helper()
	registry := registryAt(t)
	return vaults.NewAdd(vaultreader.Identity{}, registry, time.Now), registry
}

func registryAt(t *testing.T) *appstate.VaultRegistry {
	t.Helper()
	return appstate.OpenAt(filepath.Join(t.TempDir(), "state", "vaults.json"))
}

// folder makes a directory named name, under a parent of its own.
func folder(t *testing.T, name string) string {
	t.Helper()
	pos := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(pos, 0o755); err != nil {
		t.Fatal(err)
	}
	return pos
}

func TestANameAnotherVaultHasGetsANumber(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	var names []string
	for range 3 {
		v, err := add.Execute(folder(t, "notes"), "")
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, v.Name)
	}
	want := []string{"notes", "notes 2", "notes 3"}
	for i, name := range want {
		if names[i] != name {
			t.Errorf("the vaults are called %v, want %v", names, want)
			break
		}
	}

	recognized, err := registry.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(recognized) != 3 {
		t.Errorf("the list holds %d vaults, want 3", len(recognized))
	}
}

func TestANameIsTakenWhateverItsCaseAndComposition(t *testing.T) {
	t.Parallel()
	// A folder name from a file picker arrives decomposed and the same name
	// typed at a command line arrives composed.
	add, _ := newAdd(t)
	if _, err := add.Execute(folder(t, "first"), norm.NFC.String("Café")); err != nil {
		t.Fatal(err)
	}
	v, err := add.Execute(folder(t, "second"), norm.NFD.String("café"))
	if err != nil {
		t.Fatal(err)
	}
	if v.Name != norm.NFD.String("café")+" 2" {
		t.Errorf("the second vault is called %q, want the taken name numbered", v.Name)
	}
}

func TestAVaultInsideAnotherIsRefused(t *testing.T) {
	t.Parallel()
	add, _ := newAdd(t)
	outer := folder(t, "outer")
	if _, err := add.Execute(outer, ""); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(outer, "projects", "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := add.Execute(inner, ""); !errors.Is(err, domain.ErrOverlaps) {
		t.Errorf("a folder inside a vault was answered %v", err)
	}
	if _, carriesOne, err := (vaultreader.Identity{}).GetVaultID(inner); err != nil || carriesOne {
		t.Errorf("the rejected folder was given an identity: %v %v", carriesOne, err)
	}
}

func TestAVaultContainingAnotherIsRefused(t *testing.T) {
	t.Parallel()
	add, _ := newAdd(t)
	outer := folder(t, "outer")
	inner := filepath.Join(outer, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := add.Execute(inner, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := add.Execute(outer, ""); !errors.Is(err, domain.ErrOverlaps) {
		t.Errorf("a folder holding a vault was answered %v", err)
	}
}

func TestAFolderReachedThroughASymlinkIsTheFolderItself(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	target := folder(t, "notes")
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("this machine does not make symlinks: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}

	first, err := add.Execute(resolved, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := add.Execute(link, "")
	if err != nil {
		t.Fatalf("the same folder under two names: %v", err)
	}
	if second.Path != resolved {
		t.Errorf("added at %s, want %s", second.Path, resolved)
	}
	if second.ID != first.ID {
		t.Errorf("one folder carries two identities: %s and %s", first.ID, second.ID)
	}

	recognized, err := registry.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(recognized) != 1 {
		t.Errorf("the list holds %v, want one vault", recognized)
	}
}

func TestAPathThatIsNotAbsoluteIsRefused(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)

	if _, err := add.Execute(".", ""); err == nil {
		t.Fatal("a relative path was taken for a vault")
	}
	recognized, err := registry.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(recognized) != 0 {
		t.Errorf("the list holds %v", recognized)
	}
}

func TestAFolderThatIsNotThereIsUnreadable(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)

	_, err := add.Execute(filepath.Join(t.TempDir(), "missing"), "")
	if !errors.Is(err, vaults.ErrUnreadable) {
		t.Fatalf("want ErrUnreadable, got %v", err)
	}
	recognized, recognizedErr := registry.List()
	if recognizedErr != nil {
		t.Fatal(recognizedErr)
	}
	if len(recognized) != 0 {
		t.Errorf("the list holds %v", recognized)
	}
}

func TestAdd_UnreadableVaultIsNotAdded(t *testing.T) {
	t.Parallel()
	root := folder(t, "notes")
	for _, tc := range []struct {
		name     string
		identity refusingIdentity
		registry func(*appstate.VaultRegistry) refusingRegistry
	}{
		{"the list cannot be read", refusingIdentity{}, func(r *appstate.VaultRegistry) refusingRegistry {
			return refusingRegistry{VaultRegistry: r, list: errRefused}
		}},
		{"the identity cannot be given", refusingIdentity{ensure: errRefused}, func(r *appstate.VaultRegistry) refusingRegistry {
			return refusingRegistry{VaultRegistry: r}
		}},
		{"the list cannot be searched", refusingIdentity{}, func(r *appstate.VaultRegistry) refusingRegistry {
			return refusingRegistry{VaultRegistry: r, find: errRefused}
		}},
		{"the list cannot be written", refusingIdentity{}, func(r *appstate.VaultRegistry) refusingRegistry {
			return refusingRegistry{VaultRegistry: r, save: errRefused}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			inner := registryAt(t)
			add := vaults.NewAdd(tc.identity, tc.registry(inner), time.Now)

			_, err := add.Execute(root, "")
			if !errors.Is(err, errRefused) {
				t.Fatalf("want the rejection back, got %v", err)
			}
			recognized, recognizedErr := inner.List()
			if recognizedErr != nil {
				t.Fatal(recognizedErr)
			}
			if len(recognized) != 0 {
				t.Errorf("the list holds %v", recognized)
			}
		})
	}
}

// A folder copied with the identity inside it is a second folder claiming the
// first one's identity.
func TestACopyOfAVaultIsRefusedAndAMoveIsFollowed(t *testing.T) {
	t.Parallel()
	add, registry := newAdd(t)
	original := folder(t, "notes")
	first, err := add.Execute(original, "")
	if err != nil {
		t.Fatal(err)
	}

	copied := filepath.Join(t.TempDir(), "copy")
	if err := os.CopyFS(copied, os.DirFS(original)); err != nil {
		t.Fatal(err)
	}
	if _, err := add.Execute(copied, ""); !errors.Is(err, vaults.ErrCopy) {
		t.Fatalf("want ErrCopy, got %v", err)
	}
	recognized, recognizedErr := registry.List()
	if recognizedErr != nil {
		t.Fatal(recognizedErr)
	}
	if len(recognized) != 1 || recognized[0].Path != first.Path {
		t.Errorf("the list holds %v, want the original only", recognized)
	}

	moved := filepath.Join(t.TempDir(), "moved")
	if err := os.Rename(original, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(copied); err != nil {
		t.Fatal(err)
	}
	v, err := add.Execute(moved, "")
	if err != nil {
		t.Fatal(err)
	}
	if v.ID != first.ID {
		t.Errorf("the moved folder is %s, want %s", v.ID, first.ID)
	}
}

func TestAnOldLocationThatCannotBeReadStopsTheAdd(t *testing.T) {
	t.Parallel()
	registry := registryAt(t)
	first, err := vaults.NewAdd(vaultreader.Identity{}, registry, time.Now).Execute(folder(t, "old"), "")
	if err != nil {
		t.Fatal(err)
	}
	identity := refusingIdentity{
		ensureAs: first.ID,
		vaultID: func(string) (domain.VaultID, bool, error) {
			return "", false, errRefused
		},
	}

	_, err = vaults.NewAdd(identity, registry, time.Now).Execute(folder(t, "moved"), "")
	if !errors.Is(err, errRefused) {
		t.Fatalf("want the rejection back, got %v", err)
	}
}

func TestAnUnreadableFolderKeepsTheCauseInTheError(t *testing.T) {
	t.Parallel()
	add := vaults.NewAdd(refusingIdentity{checkReadable: errRefused}, registryAt(t), time.Now)

	_, err := add.Execute(folder(t, "notes"), "")
	if !errors.Is(err, vaults.ErrUnreadable) || !errors.Is(err, errRefused) {
		t.Fatalf("want both the sentinel and the cause, got %v", err)
	}
}

// A copy exists only when the old location still carries this identity.
func TestAnOldLocationThatIsNotThisVaultAnyMoreIsAMove(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		does bool
		id   domain.VaultID
	}{
		{"carries another identity", true, "01OTHER"},
		{"carries none", false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			registry := registryAt(t)
			first, err := vaults.NewAdd(vaultreader.Identity{}, registry, time.Now).Execute(folder(t, "old"), "")
			if err != nil {
				t.Fatal(err)
			}
			identity := refusingIdentity{
				ensureAs: first.ID,
				vaultID: func(string) (domain.VaultID, bool, error) {
					return tc.id, tc.does, nil
				},
			}
			moved, err := filepath.EvalSymlinks(folder(t, "moved"))
			if err != nil {
				t.Fatal(err)
			}

			v, err := vaults.NewAdd(identity, registry, time.Now).Execute(moved, "")
			if err != nil {
				t.Fatal(err)
			}
			if v.Path != moved || v.ID != first.ID {
				t.Errorf("got %+v, want the vault at %s", v, moved)
			}
		})
	}
}
