package picker_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
)

type saveDialog struct {
	path     string
	isChosen bool
	fails    error
	name     string
	accept   []string
}

func (d *saveDialog) ChooseSaveFile(_ context.Context, _, name string, accept []string) (string, bool, error) {
	d.name, d.accept = name, accept
	return d.path, d.isChosen, d.fails
}

func TestSaveTextWritesTheTextWhereThePersonChose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "account.json")
	dialog := &saveDialog{path: path, isChosen: true}

	saved, err := picker.NewSaveText(dialog).Execute(t.Context(), picker.TextFile{
		Title: "Save", Name: "account.json", Extension: ".json", Content: `{"a":1}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !saved.IsSaved || saved.Path != path {
		t.Errorf("the answer is %+v", saved)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != `{"a":1}` {
		t.Errorf("the file holds %q, %v", got, err)
	}
	if dialog.name != "account.json" || len(dialog.accept) != 1 || dialog.accept[0] != ".json" {
		t.Errorf("the dialog proposed %q and offered %v", dialog.name, dialog.accept)
	}
}

func TestSaveTextOffersEveryFileWithoutAnExtension(t *testing.T) {
	dialog := &saveDialog{}

	if _, err := picker.NewSaveText(dialog).Execute(t.Context(), picker.TextFile{Name: "notes"}); err != nil {
		t.Fatal(err)
	}
	if len(dialog.accept) != 0 {
		t.Errorf("the dialog offered %v", dialog.accept)
	}
}

func TestSaveTextWritesNothingWhenThePersonClosedTheDialog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "account.json")

	saved, err := picker.NewSaveText(&saveDialog{path: path}).Execute(t.Context(), picker.TextFile{Content: "x"})
	if err != nil || saved.IsSaved {
		t.Fatalf("the answer is %+v, %v", saved, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a file was written: %v", err)
	}
}

func TestSaveTextReportsADialogThisMachineCannotShow(t *testing.T) {
	_, err := picker.NewSaveText(&saveDialog{fails: machine.ErrNoFolderDialog}).Execute(t.Context(), picker.TextFile{})
	if !errors.Is(err, machine.ErrNoFolderDialog) {
		t.Errorf("the error is %v", err)
	}
}

func TestSaveTextReportsAFolderThatIsGone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gone", "account.json")

	saved, err := picker.NewSaveText(&saveDialog{path: path, isChosen: true}).Execute(t.Context(), picker.TextFile{})
	if err == nil || saved.IsSaved {
		t.Errorf("the answer is %+v, %v", saved, err)
	}
}
