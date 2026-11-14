package profilefiles

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

func TestTheDefaultDirectoryIsTheProfilesFolderOfTheDataDirectory(t *testing.T) {
	t.Setenv("NUMEN_PLUGIN_DATA_DIR", "/tmp/numen-data")
	if got := GetDefaultDirectory().path; got != "/tmp/numen-data/profiles" {
		t.Fatalf("directory = %q", got)
	}
	t.Setenv("NUMEN_PLUGIN_DATA_DIR", "")
	if got := GetDefaultDirectory().path; got != "" {
		t.Fatalf("directory without a data directory = %q, want empty", got)
	}
}

func writeProfile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestOnlyTheJSONFilesOfTheFolderAreRead(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "a-broken.json", `{"id": "owner/x", `)
	writeProfile(t, dir, "e-good.json", `{"id":"owner/good","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"m.onnx"}`)
	writeProfile(t, dir, "notes.txt", `not a profile`)
	if err := os.MkdirAll(filepath.Join(dir, "sub.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "missing-target"), filepath.Join(dir, "b-unreadable.json")); err != nil {
		t.Fatal(err)
	}

	reported := map[string]error{}
	table, err := NewDirectory(dir).Load(func(file string, err error) { reported[filepath.Base(file)] = err })
	if err != nil {
		t.Fatal(err)
	}
	if len(reported) != 2 || reported["a-broken.json"] == nil || !errors.Is(reported["b-unreadable.json"], fs.ErrNotExist) {
		t.Errorf("reported %v, want a-broken.json and b-unreadable.json (ErrNotExist)", reported)
	}

	if _, err := table.GetLocal("owner/good"); err != nil {
		t.Errorf("a good file next to a broken one was skipped: %v", err)
	}
	if _, err := table.GetLocal(model.DefaultModel); err != nil {
		t.Errorf("the shipped table lost a model: %v", err)
	}
}

func TestLoadKeepsShippedProfilesWhenFolderIsMissing(t *testing.T) {
	table, err := NewDirectory(filepath.Join(t.TempDir(), "profiles")).Load(func(file string, err error) {
		t.Errorf("unexpected report for missing folder: %s: %v", file, err)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := table.GetLocal(model.DefaultModel); err != nil {
		t.Error(err)
	}
}

func TestLoadKeepsShippedProfilesWhenPathIsNotFolder(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	var reportedPath string
	table, err := NewDirectory(file).Load(func(path string, _ error) { reportedPath = path })
	if err != nil {
		t.Fatal(err)
	}
	if reportedPath != file {
		t.Errorf("reportedPath = %q, want %q", reportedPath, file)
	}
	if _, err := table.GetLocal(model.DefaultModel); err != nil {
		t.Error(err)
	}
}
