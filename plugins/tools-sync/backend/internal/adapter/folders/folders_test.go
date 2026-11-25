package folders_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/folders"
)

func TestReceived_MakesTheFolderOfTheNameAndNumbersANameThatIsTaken(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Shared")
	received := folders.NewReceivedFolders(func() string { return root })

	first, err := received.MakeReceivedFolder(t.Context(), "Food")
	if err != nil {
		t.Fatal(err)
	}
	second, err := received.MakeReceivedFolder(t.Context(), "Food")
	if err != nil {
		t.Fatal(err)
	}

	if first.Name != "Food" || first.Path != filepath.Join(root, "Food") {
		t.Errorf("first = %+v, want Food in the folder of the setting", first)
	}
	if second.Name != "Food 2" || second.Path != filepath.Join(root, "Food 2") {
		t.Errorf("second = %+v, want Food 2", second)
	}
	for _, path := range []string{first.Path, second.Path} {
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Errorf("%s: no folder was made (%v)", path, err)
		}
	}
}

func TestReceived_UsesNumenSharedInTheHomeFolderWhenTheSettingIsEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	received := folders.NewReceivedFolders(func() string { return "" })

	made, err := received.MakeReceivedFolder(t.Context(), "Plans")
	if err != nil {
		t.Fatal(err)
	}

	if want := filepath.Join(home, "Numen Shared", "Plans"); made.Path != want {
		t.Errorf("path = %q, want %q", made.Path, want)
	}
}
