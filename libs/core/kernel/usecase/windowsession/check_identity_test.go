package windowsession_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func newFolderVault(t *testing.T) domain.Vault {
	t.Helper()
	root := filepath.Join(t.TempDir(), "notes")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	id, err := vaultreader.Identity{}.EnsureIdentity(root, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return domain.Vault{ID: id, Name: "notes", Path: root}
}

func TestAVaultWhoseFolderCarriesItsIdentityIsReadable(t *testing.T) {
	t.Parallel()
	v := newFolderVault(t)

	if err := windowsession.NewCheckIdentity(vaultreader.Identity{}).Execute(v); err != nil {
		t.Fatalf("a vault in its own folder failed: %v", err)
	}
}

func TestAFolderCarryingAnotherIdentityIsNotTheVault(t *testing.T) {
	t.Parallel()
	v := newFolderVault(t)
	v.ID = "another"

	if err := windowsession.NewCheckIdentity(vaultreader.Identity{}).Execute(v); !errors.Is(err, vaultdomain.ErrUnreadable) {
		t.Fatalf("error is %v, want ErrUnreadable", err)
	}
}

func TestAFolderThatIsMissingIsNotReadable(t *testing.T) {
	t.Parallel()
	v := newFolderVault(t)
	if err := os.RemoveAll(v.Path); err != nil {
		t.Fatal(err)
	}

	if err := windowsession.NewCheckIdentity(vaultreader.Identity{}).Execute(v); !errors.Is(err, vaultdomain.ErrUnreadable) {
		t.Fatalf("error is %v, want ErrUnreadable", err)
	}
}
