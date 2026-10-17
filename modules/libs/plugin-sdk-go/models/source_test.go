package models

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetDirIsTheFolderTheSettingsName(t *testing.T) {
	got, err := Source{Dir: "/srv/models"}.GetDir()
	if err != nil || got != "/srv/models" {
		t.Fatalf("GetDir() = %q, %v", got, err)
	}
}

func TestGetDirIsTheSharedModelsFolder(t *testing.T) {
	shared := t.TempDir()
	t.Setenv(EnvDir, shared)
	got, err := Source{}.GetDir()
	if err != nil || got != shared {
		t.Fatalf("GetDir() = %q, %v", got, err)
	}
}

func TestSourceFetchReturnsAFileThatIsAlreadyThere(t *testing.T) {
	dir := t.TempDir()
	address, sum := "https://example.invalid/model.onnx", "2d711642b726b04401627ca9fbac32f5c8530fb1903cc4db02258717921a4881"
	at := filepath.Join(dir, GetCacheName(address, sum))
	if err := os.WriteFile(at, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Source{Dir: dir}.Fetch(context.Background(), address, sum)
	if err != nil || got != at {
		t.Fatalf("Fetch() = %q, %v", got, err)
	}
}

func TestSourceFetchNamesTheSectionWhenDownloadsAreOff(t *testing.T) {
	_, err := Source{Dir: t.TempDir(), Section: "indexing.test"}.Fetch(context.Background(), "https://example.invalid/m.onnx", "")
	if err == nil || !strings.Contains(err.Error(), "indexing.test.download") {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestSourceFetchNamesTheFolderWhenThereIsNone(t *testing.T) {
	t.Setenv(EnvDir, "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("AppData", "")
	if _, err := (Source{Section: "indexing.test"}).Fetch(context.Background(), "https://example.invalid/m.onnx", ""); err == nil {
		t.Fatal("Fetch() found a folder on a machine that names none")
	}
}

func TestGetSearchPathsAreTheSettingsFolderAndTheInstallFolder(t *testing.T) {
	got := GetSearchPaths("/srv/models", "m.onnx")
	if len(got) == 0 || got[0] != filepath.Join("/srv/models", "m.onnx") {
		t.Fatalf("GetSearchPaths() = %v, want the settings folder first", got)
	}
	self, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	want := filepath.Join(filepath.Dir(self), "models", "m.onnx")
	found := false
	for _, one := range got {
		found = found || one == want
	}
	if !found {
		t.Errorf("GetSearchPaths() = %v lacks %s", got, want)
	}
}
