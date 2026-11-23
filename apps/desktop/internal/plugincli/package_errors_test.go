package plugincli

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var errWalkFailed = errors.New("walk failed")

func writePluginDir(t *testing.T) string {
	t.Helper()
	pluginDir := filepath.Join(t.TempDir(), "plugin")
	if err := os.MkdirAll(filepath.Join(pluginDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id": "md.numen.plugin", "name": "Plugin", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return pluginDir
}

func TestIsSamePathJoinsARelativePathToTheWorkingDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if !isSamePath("plugin.zip", filepath.Join(cwd, "plugin.zip")) {
		t.Errorf("isSamePath(plugin.zip, %q) = false, want true", filepath.Join(cwd, "plugin.zip"))
	}
	if isSamePath("plugin.zip", "plugin.zip") {
		t.Error("isSamePath(plugin.zip, plugin.zip) = true, want false while the working directory exists")
	}
}

func TestIsSamePathComparesThePathWhenTheWorkingDirectoryIsRemoved(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux reports a removed working directory as an error")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if !isSamePath("plugin.zip", "plugin.zip") {
		t.Error("isSamePath(plugin.zip, plugin.zip) = false, want the path compared unchanged")
	}
}

func TestPluginPackage_ReportsAnOutputDirectoryThatCannotBeCreated(t *testing.T) {
	pluginDir := writePluginDir(t)
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := BuildPackage(pluginDir, filepath.Join(blocker, "plugin.zip"), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "create output directory:") {
		t.Fatalf("expected create output directory error, got %v", err)
	}
}

func TestPluginPackage_ReportsAnArchiveFileThatCannotBeCreated(t *testing.T) {
	pluginDir := writePluginDir(t)
	var stdout bytes.Buffer

	_, err := BuildPackage(pluginDir, t.TempDir(), &stdout)
	if err == nil || !strings.Contains(err.Error(), "create archive file:") {
		t.Fatalf("expected create archive file error for a directory as output, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want nothing reported", stdout.String())
	}
}

func TestPluginPackage_ReportsAShippedFileThatCannotBeRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating a symbolic link needs a privilege on Windows")
	}
	pluginDir := writePluginDir(t)
	if err := os.Symlink(filepath.Join(pluginDir, "missing"), filepath.Join(pluginDir, "bin", "link")); err != nil {
		t.Fatal(err)
	}

	_, err := BuildPackage(pluginDir, filepath.Join(t.TempDir(), "plugin.zip"), &bytes.Buffer{})
	if !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "add bin/link to archive:") {
		t.Fatalf("expected add bin/link to archive error wrapping ErrNotExist, got %v", err)
	}
}

func TestArchiveWalker_ReturnsTheWalkError(t *testing.T) {
	walker := archiveWalker{archive: zip.NewWriter(io.Discard), pluginDir: "plugin", absDest: "", count: 0}
	if err := walker.Walk("bin", nil, errWalkFailed); !errors.Is(err, errWalkFailed) {
		t.Fatalf("Walk = %v, want the walk error", err)
	}
}

func TestAddFileToZip_ReportsANameTheArchiveCannotHold(t *testing.T) {
	source := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(source, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := addFileToZip(zip.NewWriter(io.Discard), source, strings.Repeat("a", 1<<16)); err == nil {
		t.Fatal("addFileToZip accepted a name longer than a zip header holds")
	}
}
