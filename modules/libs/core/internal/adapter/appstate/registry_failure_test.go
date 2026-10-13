package appstate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
)

func isolateEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
}

func TestOpenUsesTheConfigurationFolder(t *testing.T) {
	isolateEnvironment(t)
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	if runtime.GOOS != "linux" {
		t.Skip("the configuration folder is taken from XDG_CONFIG_HOME on linux only")
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
	if runtime.GOOS != "linux" {
		t.Skip("the configuration folder is unknown on linux only when HOME and XDG_CONFIG_HOME are empty")
	}
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

// operations lists every call that reads the registry file.
func operations() map[string]func(*appstate.VaultRegistry) error {
	return map[string]func(*appstate.VaultRegistry) error{
		"List": func(r *appstate.VaultRegistry) error { _, err := r.List(); return err },
		"Save": func(r *appstate.VaultRegistry) error {
			return r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"})
		},
		"Remove":       func(r *appstate.VaultRegistry) error { return r.Remove("01AAA") },
		"RecordOpened": func(r *appstate.VaultRegistry) error { return r.RecordOpened("01AAA") },
		"GetLastVault": func(r *appstate.VaultRegistry) error { _, _, err := r.GetLastVault(); return err },
		"FindVault":    func(r *appstate.VaultRegistry) error { _, _, err := r.FindVault("personal"); return err },
	}
}

func TestAnUnparsableRegistryIsReportedAndLeftAlone(t *testing.T) {
	for name, operation := range operations() {
		t.Run(name, func(t *testing.T) {
			isolateEnvironment(t)
			path := filepath.Join(t.TempDir(), "vaults.json")
			const damaged = "{ this is not json"
			if err := os.WriteFile(path, []byte(damaged), 0o644); err != nil {
				t.Fatal(err)
			}

			if err := operation(appstate.OpenAt(path)); err == nil {
				t.Fatal("a registry that cannot be parsed was accepted")
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(raw) != damaged {
				t.Errorf("the damaged file was rewritten: %s", raw)
			}
		})
	}
}

func TestARegistryThatCannotBeReadIsReported(t *testing.T) {
	for name, operation := range operations() {
		t.Run(name, func(t *testing.T) {
			isolateEnvironment(t)
			// A folder where the file should be: it exists, and it cannot be read as a file.
			path := filepath.Join(t.TempDir(), "vaults.json")
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}

			if err := operation(appstate.OpenAt(path)); err == nil {
				t.Fatal("a registry that cannot be read was accepted")
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
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
	if err := os.Mkdir(filepath.Join(parent, "probe"), 0o755); err == nil {
		t.Skip("this account creates folders where it has no leave to")
	}

	r := appstate.OpenAt(filepath.Join(parent, "state", "vaults.json"))
	if err := r.Save(domain.Vault{ID: "01AAA", Name: "personal", Path: "/notes"}); err == nil {
		t.Fatal("a save into a folder that cannot be created reported success")
	}
}
