package note_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

const managingPlugin = "md.numen.tools.sync"

func (c changeFixture) managedVault() domain.Vault {
	managed := c.vault
	managed.ManagedBy = managingPlugin
	return managed
}

func TestWriteToVaultManagedByOtherPluginIsRefused(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Milk.md": "# Milk\n"})
	managed := c.managedVault()
	window := t.Context()
	other := vaultdomain.WithCaller(t.Context(), "md.numen.tools.link")

	for name, run := range map[string]func(context.Context) error{
		"save": func(ctx context.Context) error {
			_, err := c.newWrite().Execute(ctx, managed, "Milk.md", "changed", domain.FileStat{})
			return err
		},
		"create": func(ctx context.Context) error {
			_, err := c.create().Execute(ctx, managed, note.NewNote{Title: "Bread"})
			return err
		},
		"move": func(ctx context.Context) error {
			_, err := c.move().Execute(ctx, managed, "Milk.md", "Food/Milk.md")
			return err
		},
		"remove": func(ctx context.Context) error {
			_, err := c.remove().Execute(ctx, managed, "Milk.md")
			return err
		},
		"destroy": func(ctx context.Context) error {
			_, err := c.remove().Destroy(ctx, managed, "Milk.md")
			return err
		},
		"rename": func(ctx context.Context) error {
			_, err := note.NewRename(c.moveDeps(), c.move(), nil).Execute(ctx, managed, "Milk.md", "Bread")
			return err
		},
	} {
		for caller, ctx := range map[string]context.Context{"window": window, "other plugin": other} {
			if err := run(ctx); !errors.Is(err, vaultdomain.ErrReadOnly) {
				t.Errorf("%s from the %s: error = %v, want ErrReadOnly", name, caller, err)
			}
		}
	}
	if got := c.read(t, "Milk.md"); got != "# Milk\n" {
		t.Errorf("the file is %q after refused writes, want it unchanged", got)
	}
}

func TestTheManagingPluginWritesToItsVault(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Milk.md": "# Milk\n"})
	managed := c.managedVault()
	ctx := vaultdomain.WithCaller(t.Context(), managingPlugin)

	if _, err := c.create().Execute(ctx, managed, note.NewNote{Title: "Bread"}); err != nil {
		t.Errorf("create by the managing plugin: error = %v", err)
	}
	if _, err := c.move().Execute(ctx, managed, "Milk.md", "Food/Milk.md"); err != nil {
		t.Errorf("move by the managing plugin: error = %v", err)
	}
}

func TestAnyCallerWritesToUnmanagedVault(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, nil)

	if _, err := c.create().Execute(t.Context(), c.vault, note.NewNote{Title: "Bread"}); err != nil {
		t.Errorf("create from the window: error = %v", err)
	}
}

func TestAnyCallerDestroysInUnmanagedVault(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Milk.md": "# Milk\n"})

	if _, err := c.remove().Destroy(t.Context(), c.vault, "Milk.md"); err != nil {
		t.Errorf("destroy from the window: error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(c.vault.Path, "Milk.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the file after destroy: stat error = %v, want it gone", err)
	}
}
