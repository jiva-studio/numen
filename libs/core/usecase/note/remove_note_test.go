package note_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
)

func TestRemoveNote_FolderFailsAndRestProceeds(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"dir/Heat.md": "# Heat\n", "Work.md": "# Work\n"})
	remove := note.NewRemoveNote(c.remove(), vaultreader.Readers{})

	if _, err := remove.Execute(t.Context(), c.vault, "dir", false); !errors.Is(err, note.ErrFolder) {
		t.Errorf("a folder was removed: %v", err)
	}
	if !c.exists("dir/Heat.md") {
		t.Error("the folder's note went")
	}

	trashed, err := remove.Execute(t.Context(), c.vault, "Work.md", false)
	if err != nil {
		t.Fatal(err)
	}
	if trashed.Trashed == "" || c.exists("Work.md") {
		t.Errorf("the note was not trashed: %+v", trashed)
	}

	destroyed, err := remove.Execute(t.Context(), c.vault, "dir/Heat.md", true)
	if err != nil {
		t.Fatal(err)
	}
	if destroyed.Trashed != "" || c.exists("dir/Heat.md") {
		t.Errorf("the note was not destroyed: %+v", destroyed)
	}
}

func TestRemovingWithNoFilesOpenIsRefused(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Work.md": "# Work\n"})

	if _, err := note.NewRemoveNote(c.remove(), nil).Execute(t.Context(), c.vault, "Work.md", false); err == nil {
		t.Error("a removal with no files open went ahead")
	}
}
