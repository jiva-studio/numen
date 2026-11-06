//go:build unix

package vaultreader_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

const chartPath = "library/chart.pdf"

func getInode(t *testing.T, path string) uint64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("the stat of %s carries no inode", path)
	}
	return uint64(stat.Ino)
}

func statChart(t *testing.T, root string) domain.FileStat {
	t.Helper()
	src, err := vaultreader.Open(root, vaultconfig.Options{})
	if err != nil {
		t.Fatal(err)
	}
	stat, err := src.StatFile(t.Context(), chartPath)
	if err != nil {
		t.Fatal(err)
	}
	return stat
}

func writeChart(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestStatReportsTheInodeOfTheFile(t *testing.T) {
	root := testsupport.CopyVault(t)
	target := filepath.Join(root, filepath.FromSlash(chartPath))
	writeChart(t, target, "%PDF chart of the harbour")

	stat := statChart(t, root)
	if want := domain.FileID(getInode(t, target)); stat.FileID != want || want == 0 {
		t.Fatalf("stat gives file id %d, want the inode %d", stat.FileID, want)
	}
}

func TestAFileReplacedAtTheSamePathHasAnotherFileID(t *testing.T) {
	root := testsupport.CopyVault(t)
	target := filepath.Join(root, filepath.FromSlash(chartPath))
	writeChart(t, target, "%PDF chart of the harbour")
	before := statChart(t, root)

	replacement := filepath.Join(root, "library", "chart.pdf.new")
	writeChart(t, replacement, "%PDF chart of the harbour")
	if err := os.Rename(replacement, target); err != nil {
		t.Fatal(err)
	}

	after := statChart(t, root)
	if after.FileID == before.FileID {
		t.Fatalf("a replaced file keeps the file id %d", after.FileID)
	}
	if after.FileID != domain.FileID(getInode(t, target)) {
		t.Fatalf("stat gives file id %d, want the inode of the new file %d", after.FileID, getInode(t, target))
	}
}
