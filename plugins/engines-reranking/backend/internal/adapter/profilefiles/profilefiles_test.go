package profilefiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
)

func writeProfile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTheDefaultDirectoryIsTheProfilesFolderOfTheDataDirectory(t *testing.T) {
	data := t.TempDir()
	t.Setenv("NUMEN_PLUGIN_DATA_DIR", data)
	if got := GetDefaultDirectory().path; got != filepath.Join(data, "profiles") {
		t.Fatalf("directory = %q, want the profiles folder of %s", got, data)
	}

	t.Setenv("NUMEN_PLUGIN_DATA_DIR", "")
	if got := GetDefaultDirectory().path; got != "" {
		t.Fatalf("directory without a data directory = %q, want empty", got)
	}
}

func TestLoadReportsAndSkipsAFileWithAnUnknownField(t *testing.T) {
	dir := t.TempDir()
	unknown := filepath.Join(dir, "a-pooling.json")
	writeProfile(t, dir, "a-pooling.json", `{"id":"owner/lantern-reranker","maxTokens":256,"onnxFile":"onnx/model.onnx","pooling":"cls"}`)
	writeProfile(t, dir, "b-good.json", `{"id":"owner/harbor-reranker","maxTokens":256,"onnxFile":"onnx/model.onnx"}`)
	var reported []string

	table, err := Directory{path: dir}.Load(func(file string, _ error) { reported = append(reported, file) })

	if err != nil {
		t.Fatal(err)
	}
	if len(reported) != 1 || reported[0] != unknown {
		t.Errorf("reported %v, want only %s", reported, unknown)
	}
	if _, err := table.GetLocal("owner/lantern-reranker"); err == nil {
		t.Error("the profile of the file with an unknown field is listed")
	}
	if _, err := table.GetLocal("owner/harbor-reranker"); err != nil {
		t.Errorf("a good file next to a skipped one was skipped: %v", err)
	}
	for _, p := range []string{"onnx-community/gte-multilingual-reranker-base", "onnx-community/bge-reranker-v2-m3-ONNX"} {
		if _, err := table.GetLocal(p); err != nil {
			t.Errorf("the shipped table lost %s: %v", p, err)
		}
	}
}

func TestOnlyTheJSONFilesOfTheFolderAreRead(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "notes.txt", `not a profile`)
	writeProfile(t, dir, "lantern.JSON", `{"id":"owner/lantern-reranker","maxTokens":256,"onnxFile":"onnx/model.onnx"}`)
	if err := os.MkdirAll(filepath.Join(dir, "sub.json"), 0o700); err != nil {
		t.Fatal(err)
	}

	table, err := Directory{path: dir}.Load(func(file string, err error) { t.Errorf("reported %s: %v", file, err) })

	if err != nil {
		t.Fatal(err)
	}
	if _, err := table.GetLocal("owner/lantern-reranker"); err != nil {
		t.Error(err)
	}
}

func TestLoadKeepsTheShippedProfilesWhenTheFolderIsMissing(t *testing.T) {
	table, err := Directory{path: filepath.Join(t.TempDir(), "profiles")}.Load(func(file string, err error) { t.Errorf("reported %s: %v", file, err) })

	if err != nil {
		t.Fatal(err)
	}
	if _, err := table.GetLocal(profile.DefaultModel); err != nil {
		t.Error(err)
	}
}
