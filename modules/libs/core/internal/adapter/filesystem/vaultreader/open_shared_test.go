package vaultreader_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

func TestAnOpenSharedFileReadsTheBytesAndCanBeRenamedAndRemoved(t *testing.T) {
	vault := testsupport.NewVault(t, map[string]string{"a.pdf": "the bytes"})
	reader, err := vaultreader.Readers{}.Open(vault)
	if err != nil {
		t.Fatal(err)
	}
	file, err := reader.OpenShared(t.Context(), "a.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	got, err := io.ReadAll(file)
	if err != nil || string(got) != "the bytes" {
		t.Fatalf("read %q, %v", got, err)
	}
	from := filepath.Join(vault.Path, "a.pdf")
	if err := os.Rename(from, filepath.Join(vault.Path, "b.pdf")); err != nil {
		t.Errorf("an open file could not be renamed: %v", err)
	}
	if err := os.Remove(filepath.Join(vault.Path, "b.pdf")); err != nil {
		t.Errorf("an open file could not be removed: %v", err)
	}
}
