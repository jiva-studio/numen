package appstate_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func isolateEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
}

func TestOpenUsesTheConfigurationFolder(t *testing.T) {
	isolateEnvironment(t)
	config, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}

	r, err := appstate.Open()
	if err != nil {
		t.Fatal(err)
	}
	if r == nil {
		t.Fatal("no registry")
	}
	if want := filepath.Join(config, "numen", "vaults.json"); r.Path() != want {
		t.Errorf("path = %q, want %q", r.Path(), want)
	}
}

func TestOpenFailsWithoutAConfigurationFolder(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	r, err := appstate.Open()
	if err == nil {
		t.Fatalf("opened %v with nowhere to keep it", r)
	}
	if r != nil {
		t.Errorf("registry = %v, want none", r)
	}
}

func TestPathIsWhereTheRegistryWasOpened(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vaults.json")
	if got := appstate.OpenAt(path).Path(); got != path {
		t.Errorf("path = %q, want %q", got, path)
	}
}

// operation is one call that reads the registry file; found is what it
// reports about a vault.
type operation func(*appstate.VaultRegistry) (found bool, err error)

func operations() map[string]operation {
	return map[string]operation{
		"List": func(r *appstate.VaultRegistry) (bool, error) { _, err := r.List(); return false, err },
		"Save": func(r *appstate.VaultRegistry) (bool, error) {
			return false, r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"})
		},
		"Remove":       func(r *appstate.VaultRegistry) (bool, error) { return false, r.Remove("01AAA") },
		"RecordOpened": func(r *appstate.VaultRegistry) (bool, error) { return false, r.RecordOpened("01AAA") },
		"GetLastVault": func(r *appstate.VaultRegistry) (bool, error) { _, found, err := r.GetLastVault(); return found, err },
		"FindVault": func(r *appstate.VaultRegistry) (bool, error) {
			_, found, err := r.FindVault("personal")
			return found, err
		},
	}
}

func TestAnUnparsableRegistryIsReportedAndLeftAlone(t *testing.T) {
	for name, call := range operations() {
		t.Run(name, func(t *testing.T) {
			isolateEnvironment(t)
			path := filepath.Join(t.TempDir(), "vaults.json")
			const damaged = "{ this is not json"
			if err := os.WriteFile(path, []byte(damaged), 0o644); err != nil {
				t.Fatal(err)
			}

			found, err := call(appstate.OpenAt(path))
			var syntax *json.SyntaxError
			if !errors.As(err, &syntax) {
				t.Fatalf("err = %v, want the parse failure", err)
			}
			if found {
				t.Error("a vault was reported from a registry that cannot be parsed")
			}
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(raw) != damaged {
				t.Errorf("the damaged file was rewritten: %s", raw)
			}
		})
	}
}

func TestARegistryThatCannotBeReadIsReported(t *testing.T) {
	for name, call := range operations() {
		t.Run(name, func(t *testing.T) {
			isolateEnvironment(t)
			// A folder where the file should be: it exists, and it cannot be read as a file.
			path := filepath.Join(t.TempDir(), "vaults.json")
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}

			found, err := call(appstate.OpenAt(path))
			var unreadable *fs.PathError
			if !errors.As(err, &unreadable) {
				t.Fatalf("err = %v, want the read failure", err)
			}
			if found {
				t.Error("a vault was reported from a registry that cannot be read")
			}
		})
	}
}

func TestAFileWithoutAVersionIsSavedAsVersionOne(t *testing.T) {
	isolateEnvironment(t)
	path := filepath.Join(t.TempDir(), "vaults.json")
	if err := os.WriteFile(path, []byte(`{"vaults":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	r := appstate.OpenAt(path)
	if err := r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		V int `json:"v"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.V != 1 {
		t.Errorf("version = %d, want 1: %s", parsed.V, raw)
	}
}

func TestTheFileEndsWithANewline(t *testing.T) {
	isolateEnvironment(t)
	path := filepath.Join(t.TempDir(), "vaults.json")
	if err := appstate.OpenAt(path).Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		t.Errorf("the file does not end with a newline: %q", raw)
	}
}

func TestSavingIntoAFolderThatCannotBeCreatedIsReported(t *testing.T) {
	isolateEnvironment(t)
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(parent, 0o700); err != nil {
			t.Error(err)
		}
	})
	if err := os.Mkdir(filepath.Join(parent, "probe"), 0o755); err == nil {
		t.Skip("this account creates folders where it has no leave to")
	}

	r := appstate.OpenAt(filepath.Join(parent, "state", "vaults.json"))
	err := r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"})
	var failure *fs.PathError
	if !errors.As(err, &failure) || failure.Op != "mkdir" {
		t.Fatalf("err = %v, want the failure to create the folder", err)
	}
}

func TestTheFolderAndFileAreTheOwnersAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows files have no permission bits to read")
	}
	isolateEnvironment(t)
	dir := filepath.Join(t.TempDir(), "state")
	path := filepath.Join(dir, "vaults.json")
	if err := appstate.OpenAt(path).Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err != nil {
		t.Fatal(err)
	}

	folder, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := folder.Mode().Perm(); perm != 0o700 {
		t.Errorf("folder mode = %o, want 700", perm)
	}
	file, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := file.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 600", perm)
	}
}

// A registry an older build left readable by everyone is the owner's alone
// after the next write.
func TestAWriteNarrowsTheModeOfAnExistingFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows files have no permission bits to read")
	}
	isolateEnvironment(t)
	path := filepath.Join(t.TempDir(), "vaults.json")
	if err := os.WriteFile(path, []byte(`{"v":1,"vaults":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := appstate.OpenAt(path).Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err != nil {
		t.Fatal(err)
	}
	file, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := file.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 600", perm)
	}
}

func TestRecordingTheVaultAlreadyRecordedWritesNothing(t *testing.T) {
	isolateEnvironment(t)
	path := filepath.Join(t.TempDir(), "vaults.json")
	r := appstate.OpenAt(path)
	if err := r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err != nil {
		t.Fatal(err)
	}
	if err := r.RecordOpened("01AAA"); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := r.RecordOpened("01AAA"); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Error("the file was replaced by a write that changed nothing")
	}
}

func TestNothingIsOpenedWhileTheLastIdentityIsEmpty(t *testing.T) {
	isolateEnvironment(t)
	r := newRegistry(t)
	if err := r.Save(domain.Vault{Name: "nameless identity", Path: "/notes"}); err != nil {
		t.Fatal(err)
	}
	if got, found, err := r.GetLastVault(); err != nil || found {
		t.Errorf("last = %v %v %v, want nothing", got, found, err)
	}
}
