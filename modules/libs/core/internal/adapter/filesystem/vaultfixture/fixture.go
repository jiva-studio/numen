package vaultfixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem/vaultconfig"
)

// PlaceFile puts a file in the vault behind the writer's back, for the kinds the
// writer does not create.
func PlaceFile(t *testing.T, root, path string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("theirs"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func CreateVault(t *testing.T, files map[string]string, ignore []string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	if ignore != nil {
		config := filepath.Join(root, vaultconfig.DefaultServiceDir, "config.json")
		var cfg vaultconfig.Config
		raw, err := os.ReadFile(config)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatal(err)
		}
		cfg.Ignore = ignore
		raw, _ = json.MarshalIndent(cfg, "", "  ")
		if err := os.WriteFile(config, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
