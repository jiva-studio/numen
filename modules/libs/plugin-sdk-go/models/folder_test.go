package models

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestGetDirIsTheFolderTheSettingsName(t *testing.T) {
	got, err := Folder{Dir: "/srv/models"}.GetDir()
	if err != nil || got != "/srv/models" {
		t.Fatalf("GetDir() = %q, %v", got, err)
	}
}

func TestGetDirIsTheSharedModelsFolder(t *testing.T) {
	shared := t.TempDir()
	t.Setenv(EnvDir, shared)
	got, err := Folder{}.GetDir()
	if err != nil || got != shared {
		t.Fatalf("GetDir() = %q, %v", got, err)
	}
}

func TestFolderFetchReturnsAFileThatIsAlreadyThere(t *testing.T) {
	dir := t.TempDir()
	address, sum := "https://example.invalid/model.onnx", "2d711642b726b04401627ca9fbac32f5c8530fb1903cc4db02258717921a4881"
	at := filepath.Join(dir, GetCacheName(address, sum))
	if err := os.WriteFile(at, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Folder{Dir: dir}.Fetch(context.Background(), address, sum)
	if err != nil || got != at {
		t.Fatalf("Fetch() = %q, %v", got, err)
	}
}

func TestFolderFetchNamesTheSectionWhenDownloadsAreOff(t *testing.T) {
	_, err := Folder{Dir: t.TempDir(), Section: "indexing.test"}.Fetch(context.Background(), "https://example.invalid/m.onnx", "")
	if err == nil || !strings.Contains(err.Error(), "indexing.test.download") {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestFolderFetchNamesTheFolderWhenThereIsNone(t *testing.T) {
	t.Setenv(EnvDir, "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("AppData", "")
	if _, err := (Folder{Section: "indexing.test"}).Fetch(context.Background(), "https://example.invalid/m.onnx", ""); err == nil {
		t.Fatal("Fetch() found a folder on a machine that names none")
	}
}

func TestGetPathsAreTheSettingsFolderAndTheInstallFolder(t *testing.T) {
	got := Folder{Dir: "/srv/models"}.GetPaths("m.onnx")
	if len(got) == 0 || got[0] != filepath.Join("/srv/models", "m.onnx") {
		t.Fatalf("GetPaths() = %v, want the settings folder first", got)
	}
	self, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	want := filepath.Join(filepath.Dir(self), "models", "m.onnx")
	if !slices.Contains(got, want) {
		t.Errorf("GetPaths() = %v lacks %s", got, want)
	}
	for _, at := range (Folder{}).GetPaths("m.onnx") {
		if filepath.Dir(at) == "." {
			t.Errorf("GetPaths() names %s, a folder the settings do not", at)
		}
	}
}

func TestSourceOfAFolderFindsAndFetchesThroughIt(t *testing.T) {
	dir := t.TempDir()
	source := Folder{Dir: dir, Section: "indexing.test"}.Source()
	if got := source.GetPaths("m.onnx"); len(got) == 0 || got[0] != filepath.Join(dir, "m.onnx") {
		t.Errorf("GetPaths() = %v", got)
	}
	if _, err := source.Fetch(context.Background(), "https://example.invalid/m.onnx", ""); err == nil || !strings.Contains(err.Error(), "indexing.test.download") {
		t.Errorf("Fetch() error = %v", err)
	}
}
