package wasm_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tetratelabs/wazero"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/wasm"
)

func newModule() []byte {
	return []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00,
		0x03, 0x02, 0x01, 0x00,
		0x07, 0x05, 0x01, 0x01, 0x66, 0x00, 0x00,
		0x0a, 0x04, 0x01, 0x02, 0x00, 0x0b,
	}
}

func compile(t *testing.T, compilation wazero.CompilationCache) {
	t.Helper()
	if compilation == nil {
		t.Fatal("expected non-nil compilation cache")
	}
	ctx := t.Context()
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCompilationCache(compilation))
	if _, err := runtime.CompileModule(ctx, newModule()); err != nil {
		t.Fatalf("compile: %v", err)
	}
	if err := runtime.Close(ctx); err != nil {
		t.Fatalf("close runtime: %v", err)
	}
	if err := compilation.Close(ctx); err != nil {
		t.Fatalf("close cache: %v", err)
	}
}

func listCachedFiles(t *testing.T, dir string) map[string]os.FileInfo {
	t.Helper()
	files := make(map[string]os.FileInfo)
	err := filepath.WalkDir(filepath.Join(dir, "wazero"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		info, infoErr := entry.Info()
		files[path] = info
		return infoErr
	})
	if err != nil {
		t.Fatalf("list the cache directory: %v", err)
	}
	return files
}

func TestACompiledModuleIsStoredInTheGrantedDirectory(t *testing.T) {
	dir := t.TempDir()

	compile(t, wasm.NewCompilationCache(dir))

	if len(listCachedFiles(t, dir)) == 0 {
		t.Error("the compilation left no file in the granted directory")
	}
}

func TestASecondRuntimeLoadsTheModuleFromTheCache(t *testing.T) {
	dir := t.TempDir()
	compile(t, wasm.NewCompilationCache(dir))
	first := listCachedFiles(t, dir)

	compile(t, wasm.NewCompilationCache(dir))
	second := listCachedFiles(t, dir)

	if len(second) != len(first) {
		t.Fatalf("the cache holds %d files after the second compile, %d after the first", len(second), len(first))
	}
	for path, before := range first {
		after, found := second[path]
		if !found || !after.ModTime().Equal(before.ModTime()) || after.Size() != before.Size() {
			t.Errorf("%s was written again by the second compile", path)
		}
	}
}

func TestADirectoryThatCannotBeUsedMakesTheCacheInMemory(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	compile(t, wasm.NewCompilationCache(blocked))

	if info, err := os.Stat(blocked); err != nil || info.IsDir() {
		t.Errorf("the file in the way was replaced: %v", err)
	}
}

func TestAnEmptyDirectoryMakesTheCacheInMemory(t *testing.T) {
	t.Chdir(t.TempDir())

	compile(t, wasm.NewCompilationCache(""))

	entries, err := os.ReadDir(".")
	if err != nil || len(entries) != 0 {
		t.Errorf("the working directory holds %v, %v, want nothing", entries, err)
	}
}
