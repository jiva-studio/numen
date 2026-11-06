package note_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

func TestRenameOfAFilenameNamedNoteOntoATakenFilenameAnswersTheNoteIsUnchanged(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Old.md":     "A measure of disorder.\n",
		"Entropy.md": "# Entropy\n",
	})

	renamed, err := c.rename().Execute(t.Context(), c.vault, "Old.md", "Entropy")

	if !errors.Is(err, vaultport.ErrAlreadyExists) {
		t.Fatalf("the rename answered %v, want ErrAlreadyExists", err)
	}
	if want := "Entropy.md already exists; Old.md is unchanged"; err.Error() != want {
		t.Errorf("the rename answered %q, want %q", err.Error(), want)
	}
	if renamed.Path != "Old.md" || renamed.Moved != nil {
		t.Errorf("the rename answered %+v, want the note at Old.md and no move", renamed)
	}
	if body := c.read(t, "Old.md"); body != "A measure of disorder.\n" {
		t.Errorf("Old.md reads %q after the refused rename", body)
	}
}
