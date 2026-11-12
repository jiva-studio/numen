package note_test

import (
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"testing"
)

func TestRenameOntoTakenFilenameReturnsTheNoteUnchanged(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{
		"Old.md":     "A measure of disorder.\n",
		"Entropy.md": "# Entropy\n",
	})

	renamed, err := c.rename().Execute(t.Context(), c.vault, "Old.md", "Entropy")

	if !errors.Is(err, vault.ErrAlreadyExists) {
		t.Fatalf("the rename returned %v, want ErrAlreadyExists", err)
	}
	if want := "Entropy.md already exists; Old.md is unchanged"; err.Error() != want {
		t.Errorf("the rename returned %q, want %q", err.Error(), want)
	}
	if renamed.Path != "Old.md" || renamed.Moved != nil {
		t.Errorf("the rename returned %+v, want the note at Old.md and no move", renamed)
	}
	if body := c.read(t, "Old.md"); body != "A measure of disorder.\n" {
		t.Errorf("Old.md reads %q after the rejected rename", body)
	}
}
