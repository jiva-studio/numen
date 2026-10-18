package onnxruntime

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

// writeConfig writes a loader configuration file and returns its path.
func writeConfig(t *testing.T, at, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return at
}

func TestReadLoaderConfigFollowsIncludesAndSkipsComments(t *testing.T) {
	root := t.TempDir()
	main := writeConfig(t, filepath.Join(root, "ld.so.conf"),
		"# the loader's folders\n/usr/local/lib\ninclude ld.so.conf.d/*.conf\nhwcap 0 nosegneg\n")
	writeConfig(t, filepath.Join(root, "ld.so.conf.d", "cuda.conf"), "/usr/local/cuda/lib64 # the toolkit\n")
	writeConfig(t, filepath.Join(root, "ld.so.conf.d", "onnx.conf"), "/opt/onnxruntime-gpu/lib,/opt/extra:/opt/more\r\n\n")
	writeConfig(t, filepath.Join(root, "ld.so.conf.d", "loop.conf"), "include "+main+"\n")

	got := readLoaderConfig(main, map[string]bool{})
	want := []string{"/usr/local/lib", "/usr/local/cuda/lib64", "/opt/onnxruntime-gpu/lib", "/opt/extra", "/opt/more"}
	if !slices.Equal(got, want) {
		t.Errorf("readLoaderConfig() = %q, want %q", got, want)
	}
}

func TestReadLoaderConfigOfAMissingFileIsEmpty(t *testing.T) {
	if got := readLoaderConfig(filepath.Join(t.TempDir(), "absent.conf"), map[string]bool{}); len(got) != 0 {
		t.Errorf("readLoaderConfig() = %q, want nothing", got)
	}
}

func TestGetLoaderDirsReadsTheSearchPathsOfDarwinAndWindows(t *testing.T) {
	sep := string(os.PathListSeparator)
	t.Setenv("DYLD_LIBRARY_PATH", "/opt/a"+sep+"/opt/b")
	t.Setenv("DYLD_FALLBACK_LIBRARY_PATH", "/opt/c")
	t.Setenv("HOME", "/srv/home")
	if got := getLoaderDirs("darwin"); len(got) != 4 || !slices.Equal(got[:3], []string{"/opt/a", "/opt/b", "/opt/c"}) {
		t.Errorf("getLoaderDirs(darwin) = %q", got)
	}

	t.Setenv("SystemRoot", "/srv/windows")
	t.Setenv("PATH", "/opt/gpu"+sep+"/opt/tools")
	want := []string{filepath.Join("/srv/windows", "System32"), "/srv/windows", "/opt/gpu", "/opt/tools"}
	if got := getLoaderDirs("windows"); !slices.Equal(got, want) {
		t.Errorf("getLoaderDirs(windows) = %q, want %q", got, want)
	}
}

func TestRemoveDuplicatesKeepsTheFirstPathToEachFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "libonnxruntime.so.1.23.0")
	if err := os.WriteFile(file, []byte("a library"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "libonnxruntime.so")
	if err := os.Symlink(file, link); err != nil {
		t.Skipf("symlink: %v", err)
	}
	other := filepath.Join(dir, "absent.so")
	if got := removeDuplicates([]string{link, file, other, other}); !slices.Equal(got, []string{link, other}) {
		t.Errorf("removeDuplicates() = %q", got)
	}
}

func TestCheckLibraryRefusesAFileThatIsNotALibrary(t *testing.T) {
	switch runtime.GOOS {
	case "darwin", "freebsd", "linux", "netbsd", "windows":
	default:
		t.Skipf("%s has no loader to open a library through", runtime.GOOS)
	}
	at := filepath.Join(t.TempDir(), getLibraryName("linux"))
	if err := os.WriteFile(at, []byte("not a library"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := checkLibrary(at); err == nil {
		t.Error("checkLibrary() opened a file that is not a library")
	}
}

func TestFindInDirsSearchesOnlyAbsoluteFolders(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "libonnxruntime.so")
	if err := os.WriteFile(at, []byte("a library"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if got := findInDirs([]string{"", ".", dir}, "libonnxruntime.so"); !slices.Equal(got, []string{at}) {
		t.Errorf("findInDirs() = %q, want only %q", got, at)
	}
}
