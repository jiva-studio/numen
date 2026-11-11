package container_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

// Reading a vault is work, and work is one entry in the list of operations. Each
// pass removes its entry when it ends, so a fully read vault reports no operations.
func TestAVaultReportsNoOperationsOnceItIsRead(t *testing.T) {
	root := t.TempDir()
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, testsupport.GetEpoch()); err != nil {
		t.Fatal(err)
	}
	cfg := container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db"), RegistryPath: filepath.Join(t.TempDir(), "vaults.json")},
	})
	if _, err := cfg.AddVault(root, "alpha"); err != nil {
		t.Fatal(err)
	}
	db, err := container.OpenIndex(cfg, t.Context())
	if err != nil {
		t.Fatal(err)
	}
	installation, err := container.OpenWindowWith(cfg, t.Context(), container.WindowOpening{DB: db, Out: os.Stderr})
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	testsupport.WaitFor(t, func() bool { return len(installation.State.Window.Tasks.List()) == 0 })
}
