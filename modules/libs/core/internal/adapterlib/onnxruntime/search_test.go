package onnxruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// writeFile puts a file with the contents given in a folder.
func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	at := filepath.Join(dir, name)
	if err := os.WriteFile(at, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return at
}

// publishLibrary makes the contents given the library published for this
// platform.
func publishLibrary(t *testing.T, contents string) {
	t.Helper()
	key := runtime.GOOS + "/" + runtime.GOARCH
	before, had := releases[key]
	sum := sha256.Sum256([]byte(contents))
	releases[key] = release{library: hex.EncodeToString(sum[:])}
	t.Cleanup(func() {
		if had {
			releases[key] = before
		} else {
			delete(releases, key)
		}
	})
}

// A library in the fetch directory is offered only where it carries the
// published sum; the installed ones follow, and the loader's own name is last.
func TestTheCandidatesAreTheLibrariesThisMachineHolds(t *testing.T) {
	publishLibrary(t, "published")
	fetched := t.TempDir()
	so := writeFile(t, fetched, "libonnxruntime.so", "published")
	writeFile(t, fetched, "libonnxruntime.dylib", "tampered")
	dll := writeFile(t, fetched, "onnxruntime.dll", "published")

	installed := t.TempDir()
	held := writeFile(t, installed, runtimeName()+".1.23", "installed")
	t.Setenv("LD_LIBRARY_PATH", installed)

	got := getLibraryCandidates(Settings{Dir: fetched})
	if len(got) < 6 {
		t.Fatalf("offered %q", got)
	}
	want := []string{so, dll, so, dll, held}
	if !slices.Equal(got[:5], want) {
		t.Errorf("offered %q first, want %q", got[:5], want)
	}
	if got[len(got)-1] != runtimeName() {
		t.Errorf("offered %q last, want %q", got[len(got)-1], runtimeName())
	}
	if slices.Contains(got, filepath.Join(fetched, "libonnxruntime.dylib")) {
		t.Errorf("a library without the published sum was offered: %q", got)
	}
}

func TestOnlyALibraryThatIsThereIsUsable(t *testing.T) {
	publishLibrary(t, "published")
	dir := t.TempDir()
	elsewhere := t.TempDir()
	cases := []struct {
		name string
		dir  string
		at   string
		want bool
	}{
		{"missing", dir, filepath.Join(dir, "absent.so"), false},
		{"missing with no fetch directory", "", filepath.Join(elsewhere, "absent.so"), false},
		{"fetched with the published sum", dir, writeFile(t, dir, "a.so", "published"), true},
		{"fetched with another sum", dir, writeFile(t, dir, "b.so", "other"), false},
		{"installed elsewhere", dir, writeFile(t, elsewhere, "c.so", "other"), true},
		{"found with no fetch directory", "", writeFile(t, elsewhere, "d.so", "other"), true},
	}
	for _, c := range cases {
		if got := isUsableLibrary(c.dir, c.at); got != c.want {
			t.Errorf("%s: usable = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAFetchedLibraryOfAPlatformWithNoReleaseIsNotUsable(t *testing.T) {
	publishLibrary(t, "published")
	delete(releases, runtime.GOOS+"/"+runtime.GOARCH)
	dir := t.TempDir()
	if isUsableLibrary(dir, writeFile(t, dir, "a.so", "published")) {
		t.Error("a library was usable on a platform nothing is published for")
	}
}

// Every candidate that does not open is reported with why, in the order given,
// and none of them is the engine.
func TestEveryLibraryThatDoesNotOpenIsRefused(t *testing.T) {
	dir := t.TempDir()
	garbage := writeFile(t, dir, "garbage.so", "not a library")
	missing := filepath.Join(dir, "missing.so")

	engine, at, refused := loadFirstLibrary([]string{garbage, missing})
	if engine != nil || at != "" {
		t.Fatalf("opened %q", at)
	}
	if len(refused) != 2 || !strings.HasPrefix(refused[0], garbage+": ") || !strings.HasPrefix(refused[1], missing+": ") {
		t.Errorf("refused %q", refused)
	}

	if engine, at, refused := loadFirstLibrary(nil); engine != nil || at != "" || len(refused) != 0 {
		t.Errorf("with nothing to open: %v, %q, %q", engine, at, refused)
	}
}

func TestPathsAreTheSettingsFolderAndTheApplicationsOwn(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	here := filepath.Dir(self)

	got := GetPaths("settings", "a", "b")
	want := []string{
		filepath.Join("settings", "a"), filepath.Join(here, "a"), filepath.Join(here, "models", "a"),
		filepath.Join("settings", "b"), filepath.Join(here, "b"), filepath.Join(here, "models", "b"),
	}
	if !slices.Equal(got, want) {
		t.Errorf("paths %q, want %q", got, want)
	}

	got = GetPaths("", "a")
	want = []string{filepath.Join(here, "a"), filepath.Join(here, "models", "a")}
	if !slices.Equal(got, want) {
		t.Errorf("with no settings folder: %q, want %q", got, want)
	}

	if got := GetPaths("settings"); len(got) != 0 {
		t.Errorf("with no names: %q", got)
	}
}

func TestTheDirectoriesStartWithTheLoadersOwnPathAndTheUsersProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	first, second := t.TempDir(), t.TempDir()
	t.Setenv("LD_LIBRARY_PATH", first+string(os.PathListSeparator)+second)

	got := getLibraryDirectories()
	head := []string{first, second, filepath.Join(home, ".nix-profile", "lib")}
	if len(got) < len(head)+2 || !slices.Equal(got[:3], head) {
		t.Fatalf("directories %q", got)
	}
	if got[len(got)-1] != "/lib" || !slices.Contains(got, "/usr/lib") || !slices.Contains(got, "/run/current-system/sw/lib") {
		t.Errorf("directories %q", got)
	}
}

func TestTheDirectoriesHoldNoProfileWhereThereIsNoHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("LD_LIBRARY_PATH", "")
	for _, dir := range getLibraryDirectories() {
		if strings.Contains(dir, ".nix-profile") {
			t.Errorf("directories hold %q", dir)
		}
	}
}

// Every folder on the path is searched, in order.
func TestAnInstalledLibraryIsFoundOnEveryFolderOfThePath(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	a := writeFile(t, first, "libprobe.so.1", "a")
	b := writeFile(t, second, "libprobe.so.2", "b")
	t.Setenv("LD_LIBRARY_PATH", first+string(os.PathListSeparator)+second)

	got := findInstalled("libprobe.so", "*-probe-*")
	if len(got) < 2 || !slices.Equal(got[:2], []string{a, b}) {
		t.Errorf("found %q, want %q first", got, []string{a, b})
	}
}

func TestOnlyRegularFilesAreMatched(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a1", "x")
	b := writeFile(t, dir, "a2", "x")
	if err := os.Mkdir(filepath.Join(dir, "a3"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := findMatching(filepath.Join(dir, "a*")); !slices.Equal(got, []string{a, b}) {
		t.Errorf("matched %q, want %q", got, []string{a, b})
	}
	if got := findMatching(filepath.Join(dir, "z*")); len(got) != 0 {
		t.Errorf("matched %q where nothing is named", got)
	}
	if got := findMatching(filepath.Join(dir, "[a")); len(got) != 0 {
		t.Errorf("matched %q by a pattern that is none", got)
	}
}

// With no fetch directory, a library beside the working directory is the
// machine's own, whatever it carries.
func TestALibraryBesideTheWorkingDirectoryIsUsableWithNoFetchDirectory(t *testing.T) {
	publishLibrary(t, "published")
	dir := t.TempDir()
	writeFile(t, dir, "local.so", "other")
	t.Chdir(dir)
	if !isUsableLibrary("", "local.so") {
		t.Error("a library the machine holds was refused")
	}
}
