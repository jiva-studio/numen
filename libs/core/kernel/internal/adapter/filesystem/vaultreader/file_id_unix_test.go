//go:build unix

package vaultreader_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
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
		t.Fatalf("the stat of %s has no inode", path)
	}
	return uint64(stat.Ino)
}

func statFixture(t *testing.T, root string) domain.FileStat {
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

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestStatReportsTheInodeOfTheFile(t *testing.T) {
	root := vaultfixture.CopyVault(t)
	target := filepath.Join(root, filepath.FromSlash(chartPath))
	writeFixture(t, target, "%PDF chart of the harbor")

	stat := statFixture(t, root)
	if want := domain.FileID(getInode(t, target)); stat.FileID != want || want == 0 {
		t.Fatalf("stat gives file id %d, want the inode %d", stat.FileID, want)
	}
}

func TestAFileReplacedAtTheSamePathHasAnotherFileID(t *testing.T) {
	root := vaultfixture.CopyVault(t)
	target := filepath.Join(root, filepath.FromSlash(chartPath))
	writeFixture(t, target, "%PDF chart of the harbor")
	before := statFixture(t, root)

	replacement := filepath.Join(root, "library", "chart.pdf.new")
	writeFixture(t, replacement, "%PDF chart of the harbor")
	if err := os.Rename(replacement, target); err != nil {
		t.Fatal(err)
	}

	after := statFixture(t, root)
	if after.FileID == before.FileID {
		t.Fatalf("a replaced file keeps the file id %d", after.FileID)
	}
	if after.FileID != domain.FileID(getInode(t, target)) {
		t.Fatalf("stat gives file id %d, want the inode of the new file %d", after.FileID, getInode(t, target))
	}
}
