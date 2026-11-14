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
	"testing/fstest"
)

var (
	errStatFailed  = errors.New("stat failed")
	errCloseFailed = errors.New("close failed")
	errWalkFailed  = errors.New("walk failed")
)

type openFunc func(name string) (fs.File, error)

func (f openFunc) Open(name string) (fs.File, error) { return f(name) }

type faultyFile struct {
	fs.File
	statErr  error
	closeErr error
}

func (f faultyFile) Stat() (fs.FileInfo, error) {
	if f.statErr != nil {
		return nil, f.statErr
	}
	return f.File.Stat()
}

func (f faultyFile) Close() error {
	return errors.Join(f.File.Close(), f.closeErr)
}

func openFaultyFile(t *testing.T, file faultyFile) fs.FS {
	t.Helper()
	return openFunc(func(string) (fs.File, error) {
		opened, err := fstest.MapFS{"manifest.json": {Data: []byte("{}")}}.Open("manifest.json")
		if err != nil {
			t.Fatal(err)
		}
		file.File = opened
		return file, nil
	})
}

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

func TestGetAbsPathJoinsARelativePathToTheWorkingDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := getAbsPath("plugin.zip"), filepath.Join(cwd, "plugin.zip"); got != want {
		t.Errorf("getAbsPath(plugin.zip) = %q, want %q", got, want)
	}
}

func TestGetAbsPathReturnsThePathWhenTheWorkingDirectoryIsRemoved(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux reports a removed working directory as an error")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if got := getAbsPath("plugin.zip"); got != "plugin.zip" {
		t.Errorf("getAbsPath(plugin.zip) = %q, want the path unchanged", got)
	}
}

func TestPluginPackage_ReportsAnOutputDirectoryThatCannotBeCreated(t *testing.T) {
	pluginDir := writePluginDir(t)
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Package(pluginDir, filepath.Join(blocker, "plugin.zip"), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "create output directory:") {
		t.Fatalf("expected create output directory error, got %v", err)
	}
}

func TestPluginPackage_ReportsAnArchiveThatCannotBeWritten(t *testing.T) {
	pluginDir := writePluginDir(t)
	var stdout bytes.Buffer

	_, err := Package(pluginDir, t.TempDir(), &stdout)
	if err == nil || !strings.Contains(err.Error(), "write archive file:") {
		t.Fatalf("expected write archive file error for a directory as output, got %v", err)
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
	archivePath := filepath.Join(t.TempDir(), "plugin.zip")

	_, err := Package(pluginDir, archivePath, &bytes.Buffer{})
	if !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "add bin/link to archive:") {
		t.Fatalf("expected add bin/link to archive error wrapping ErrNotExist, got %v", err)
	}
	if _, err := os.Stat(archivePath); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("archive file exists after a failed package: %v", err)
	}
}

func TestArchiveWalker_ReturnsTheWalkError(t *testing.T) {
	walker := archiveWalker{archive: zip.NewWriter(io.Discard), files: fstest.MapFS{}, pluginDir: "plugin", absDest: "", count: 0}
	if err := walker.Walk("bin", nil, errWalkFailed); !errors.Is(err, errWalkFailed) {
		t.Fatalf("Walk = %v, want the walk error", err)
	}
}

func TestWriteArchive_ReportsAnArchiveThatCannotBeFinished(t *testing.T) {
	walker := archiveWalker{
		archive:   zip.NewWriter(failingWriter{}),
		files:     fstest.MapFS{"manifest.json": {Data: []byte("{}")}},
		pluginDir: "plugin",
		absDest:   "",
		count:     0,
	}
	err := walker.writeArchive([]string{"manifest.json"})
	if !errors.Is(err, errWriteFailed) || !strings.Contains(err.Error(), "finish archive:") {
		t.Fatalf("writeArchive = %v, want a finish archive error", err)
	}
}

func TestAddFileToZip_ReportsAFileThatCannotBeStated(t *testing.T) {
	files := openFaultyFile(t, faultyFile{File: nil, statErr: errStatFailed, closeErr: nil})
	if err := addFileToZip(zip.NewWriter(io.Discard), files, "manifest.json"); !errors.Is(err, errStatFailed) {
		t.Fatalf("addFileToZip = %v, want the stat error", err)
	}
}

func TestAddFileToZip_ReportsAFileThatCannotBeClosed(t *testing.T) {
	files := openFaultyFile(t, faultyFile{File: nil, statErr: nil, closeErr: errCloseFailed})
	if err := addFileToZip(zip.NewWriter(io.Discard), files, "manifest.json"); !errors.Is(err, errCloseFailed) {
		t.Fatalf("addFileToZip = %v, want the close error", err)
	}
}

func TestAddFileToZip_ReportsANameTheArchiveCannotHold(t *testing.T) {
	files := openFaultyFile(t, faultyFile{File: nil, statErr: nil, closeErr: nil})
	if err := addFileToZip(zip.NewWriter(io.Discard), files, strings.Repeat("a", 1<<16)); err == nil {
		t.Fatal("addFileToZip accepted a name longer than a zip header holds")
	}
}
