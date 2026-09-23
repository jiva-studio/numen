package appstate_test

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
)

// What was worked out is kept by the vault it was worked out for, and comes
// back as it was written.
func TestSchedulesAreKeptByVault(t *testing.T) {
	kept := appstate.SchedulesAt(filepath.Join(t.TempDir(), "flashcards"))
	ctx := t.Context()

	if err := kept.Write(ctx, "01J8F3K2M9QRSTVWXYZ012", []byte(`{"v":1}`)); err != nil {
		t.Fatal(err)
	}
	got, err := kept.Read(ctx, "01J8F3K2M9QRSTVWXYZ012")
	if err != nil || string(got) != `{"v":1}` {
		t.Errorf("read back %q, %v", got, err)
	}

	if _, err := kept.Read(ctx, "01J8F3K2M9QRSTVWXYZ999"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a vault nothing was kept for answers %v", err)
	}
}

// The identity is written into a file's name, so a name that is not one is
// refused: anything else could reach a file this folder does not hold.
func TestAnIdentityThatIsNotOneNamesNoFile(t *testing.T) {
	kept := appstate.SchedulesAt(filepath.Join(t.TempDir(), "flashcards"))
	for _, id := range []domain.VaultID{"", "../numen.json", "a/b", `a\b`, "one.json"} {
		if err := kept.Write(t.Context(), id, []byte("{}")); err == nil {
			t.Errorf("wrote under %q", id)
		}
	}
}
