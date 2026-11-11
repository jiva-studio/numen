package container_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func newVaultConfig(t *testing.T) container.Config {
	t.Helper()
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
	return cfg
}

func gateWalk(start windowsession.VaultPassesFunc, release <-chan struct{}) windowsession.VaultPassesFunc {
	return func(
		ctx context.Context, v domain.Vault, rebuild bool, report windowsession.Reporters,
	) windowsession.Started {
		started := start(ctx, v, rebuild, report)
		walk := started.Read
		started.Read = func(ctx context.Context, during func()) (int, error) {
			select {
			case <-release:
				return walk(ctx, during)
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		}
		return started
	}
}

// Reading a vault is work, and work is one entry in the list of operations. A
// pass that reports itself elsewhere creates a second entry for the same work,
// with a separate count.
func TestAVaultReturnsWhatItIsDoingWhileItReadsItself(t *testing.T) {
	cfg := newVaultConfig(t)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseWalk := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseWalk)

	db, err := container.OpenIndex(cfg, t.Context())
	if err != nil {
		t.Fatal(err)
	}
	installation, err := container.OpenWindowHoldingWalk(cfg, t.Context(), container.WindowOpening{DB: db, Out: os.Stderr},
		func(start windowsession.VaultPassesFunc) windowsession.VaultPassesFunc {
			return gateWalk(start, release)
		})
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	// Reported before the reading goroutine starts, so a window opening on a new
	// vault shows the walk from its start.
	list := installation.State.Window.Tasks.List()
	if len(list) != 1 {
		t.Errorf("a vault just opened is doing %+v, want the walk of its notes", list)
	}
	// Not requested by the user, so it is displayed only once it has lasted.
	for _, item := range list {
		if item.IsUserStarted {
			t.Errorf("%q says a person asked for it", item.Title)
		}
	}

	// Each pass removes its entry when it ends, so a fully read vault reports no
	// operations.
	releaseWalk()
	testsupport.WaitFor(t, func() bool { return len(installation.State.Window.Tasks.List()) == 0 })
}
