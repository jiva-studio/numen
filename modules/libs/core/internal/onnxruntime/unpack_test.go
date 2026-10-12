package onnxruntime

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/iotest"
)

// createArchive is an archive laid out the way a mac release is: the library carrying
// its version, the plain name beside it as a link, and another library named for
// what it adds. What comes back is where it is and the sums it and the library
// inside it carry.
func createArchive(t *testing.T, dir string) release {
	t.Helper()
	at := filepath.Join(dir, "onnxruntime-osx-arm64-1.23.0.tgz")
	file, err := os.Create(at)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	const body = "the library"
	zipped := gzip.NewWriter(file)
	writing := tar.NewWriter(zipped)
	const under = "onnxruntime-osx-arm64-1.23.0/lib/"
	for _, one := range []struct {
		name string
		kind byte
		link string
		body string
	}{
		{name: under + "libonnxruntime_providers_shared.dylib", kind: tar.TypeReg, body: "another library"},
		{name: under + "libonnxruntime.dylib", kind: tar.TypeSymlink, link: "libonnxruntime.1.23.0.dylib"},
		{name: under + "libonnxruntime.1.23.0.dylib", kind: tar.TypeReg, body: body},
	} {
		head := &tar.Header{
			Name: one.name, Typeflag: one.kind, Linkname: one.link,
			Mode: 0o755, Size: int64(len(one.body)),
		}
		if err := writing.WriteHeader(head); err != nil {
			t.Fatal(err)
		}
		if _, err := writing.Write([]byte(one.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writing.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipped.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	return release{archive: at, sum: sum(t, at), library: computeSum(body)}
}

// sum is what one file on this machine carries.
func sum(t *testing.T, at string) string {
	t.Helper()
	held, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	return computeSum(string(held))
}

func computeSum(body string) string {
	held := sha256.Sum256([]byte(body))
	return hex.EncodeToString(held[:])
}

// A mac library carries its version before its extension, and the plain name
// beside it is a link to it. What is taken out of the archive is the file.
func TestTheMacLibraryIsTakenOutOfTheArchive(t *testing.T) {
	dir := t.TempDir()
	found := createArchive(t, dir)
	at, err := unpack(found.archive, "libonnxruntime.dylib", found)
	if err != nil {
		t.Fatal(err)
	}
	held, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if string(held) != "the library" {
		t.Errorf("what came out of the archive was %q", held)
	}
}

// An archive that is not the one published is not opened at all: nothing is
// taken out of it, and what a person is told names the file.
func TestAnArchiveThatIsNotTheOnePublishedIsRefused(t *testing.T) {
	dir := t.TempDir()
	found := createArchive(t, dir)
	found.sum = computeSum("another archive")

	at, err := unpack(found.archive, "libonnxruntime.dylib", found)
	if err == nil {
		t.Fatalf("a library was taken out of it, at %s", at)
	}
	if _, err := os.Stat(filepath.Join(dir, "libonnxruntime.dylib")); err == nil {
		t.Error("a library was left behind")
	}
}

// A cached archive that is not the one published is taken away, so the next run
// fetches it again. What stands in for that run here is a second unpack, of the
// archive as it is published.
func TestACachedArchiveThatIsNotTheOnePublishedIsFetchedAgain(t *testing.T) {
	dir := t.TempDir()
	found := createArchive(t, dir)
	broken := found
	broken.sum = computeSum("another archive")

	if _, err := unpack(broken.archive, "libonnxruntime.dylib", broken); err == nil {
		t.Fatal("an archive that is not the one published was unpacked")
	}
	if _, err := os.Stat(found.archive); err == nil {
		t.Fatal("the archive is still in the cache, and every later run is handed it")
	}

	found = createArchive(t, dir)
	at, err := unpack(found.archive, "libonnxruntime.dylib", found)
	if err != nil {
		t.Fatal(err)
	}
	held, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if string(held) != "the library" {
		t.Errorf("what came out of the archive fetched again was %q", held)
	}
}

// An archive read to its end and holding no such library says that, and not
// that it is broken. The two send a person to different places.
func TestAnArchiveHoldingNoSuchLibrarySaysSo(t *testing.T) {
	dir := t.TempDir()
	found := createArchive(t, dir)

	_, err := unpack(found.archive, "libsomethingelse.dylib", found)
	if err == nil {
		t.Fatal("a library nothing published was taken out of the archive")
	}
	if !strings.Contains(err.Error(), "holds no libsomethingelse.dylib") {
		t.Errorf("an archive holding no such library was refused as %v", err)
	}
}

// A library dropped into the cache under the name a fetched one is kept under
// is not the library this release publishes, and is written over by the one
// that is.
func TestALibraryInTheCacheThatIsNotTheOnePublishedIsWrittenOver(t *testing.T) {
	dir := t.TempDir()
	found := createArchive(t, dir)
	at := filepath.Join(dir, "libonnxruntime.dylib")
	if err := os.WriteFile(at, []byte("something else"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := unpack(found.archive, "libonnxruntime.dylib", found); err != nil {
		t.Fatal(err)
	}
	held, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if string(held) != "the library" {
		t.Errorf("what stands in the cache is %q", held)
	}
}

func isolate(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	return t.TempDir()
}

// A library that is not the one published is not left in the cache.
func TestALibraryThatIsNotTheOnePublishedIsTakenAway(t *testing.T) {
	dir := isolate(t)
	found := createArchive(t, dir)
	found.library = computeSum("another library")

	if at, err := unpack(found.archive, "libonnxruntime.dylib", found); err == nil {
		t.Fatalf("a library that is not the one published was handed back at %s", at)
	}
	if _, err := os.Stat(filepath.Join(dir, "libonnxruntime.dylib")); err == nil {
		t.Error("the library was left in the cache")
	}
}

func TestADirectoryIsNotAFileCarryingASum(t *testing.T) {
	dir := isolate(t)
	err := verify(dir, computeSum(""))
	if err == nil || !strings.Contains(err.Error(), "is not a file") {
		t.Errorf("verify a directory = %v", err)
	}
}

func TestAFileThatCannotBeOpenedIsNotVerified(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("file modes do not keep this process out")
	}
	dir := isolate(t)
	at := filepath.Join(dir, "locked")
	if err := os.WriteFile(at, []byte("body"), 0o000); err != nil {
		t.Fatal(err)
	}
	if err := verify(at, computeSum("body")); !errors.Is(err, fs.ErrPermission) {
		t.Errorf("verify a file that cannot be opened = %v", err)
	}
}

func TestAFileCarryingAnotherSumIsNotVerified(t *testing.T) {
	dir := isolate(t)
	at := filepath.Join(dir, "lib")
	if err := os.WriteFile(at, []byte("body"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verify(at, computeSum("body")); err != nil {
		t.Errorf("verify the file with its own sum = %v", err)
	}
	if err := verify(at, computeSum("other")); err == nil {
		t.Error("a file carrying another sum was verified")
	}
}

func TestAnArchiveThatIsNotThereIsNotUnpacked(t *testing.T) {
	dir := isolate(t)
	at := filepath.Join(dir, "lib")
	if err := fromTgz(filepath.Join(dir, "missing.tgz"), "lib", at); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("unpack a tgz that is not there = %v", err)
	}
	if err := fromZip(filepath.Join(dir, "missing.zip"), "lib", at); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("unpack a zip that is not there = %v", err)
	}
}

func TestAFileThatIsNotGzipIsRefusedByName(t *testing.T) {
	dir := isolate(t)
	archive := filepath.Join(dir, "plain.tgz")
	if err := os.WriteFile(archive, []byte("not gzip at all"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := fromTgz(archive, "lib", filepath.Join(dir, "lib"))
	if err == nil || !strings.Contains(err.Error(), archive) {
		t.Errorf("unpack a file that is not gzip = %v", err)
	}
}

func TestAGzipOfNoArchiveIsRefused(t *testing.T) {
	dir := isolate(t)
	archive := filepath.Join(dir, "garbage.tgz")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	zipped := gzip.NewWriter(file)
	if _, err := zipped.Write(bytes.Repeat([]byte{0xff}, 1024)); err != nil {
		t.Fatal(err)
	}
	if err := zipped.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	err = fromTgz(archive, "lib", filepath.Join(dir, "lib"))
	if err == nil || strings.Contains(err.Error(), "holds no") {
		t.Errorf("unpack a gzip holding no archive = %v", err)
	}
}

func TestAFileCannotBeWrittenWhereThereIsNoFolder(t *testing.T) {
	dir := isolate(t)
	err := write(filepath.Join(dir, "missing", "lib"), strings.NewReader("body"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("write into a folder that is not there = %v", err)
	}
}

func TestAFileThatCannotBeReadIsNotLeftHalfWritten(t *testing.T) {
	dir := isolate(t)
	at := filepath.Join(dir, "lib")
	broken := errors.New("the archive ended badly")

	if err := write(at, io.MultiReader(strings.NewReader("half"), iotest.ErrReader(broken))); !errors.Is(err, broken) {
		t.Fatalf("write from a reader that failed = %v", err)
	}
	for _, left := range []string{at, at + ".part"} {
		if _, err := os.Stat(left); err == nil {
			t.Errorf("%s was left behind", left)
		}
	}
}

// Whether something is the library by the name an archive gives it.
func TestAnArchiveNameIsTheLibraryOnlyByItsOwnName(t *testing.T) {
	for _, tc := range []struct {
		found, name string
		want        bool
	}{
		{"libonnxruntime.dylib", "libonnxruntime.dylib", true},
		{"libonnxruntime.so.1.23.0", "libonnxruntime.so", true},
		{"libonnxruntime.1.23.0.dylib", "libonnxruntime.dylib", true},
		{"onnxruntime", "onnxruntime", true},
		{"onnxruntime.1", "onnxruntime", true},
		{"onnxruntime2", "onnxruntime", false},
		{"libonnxruntime_providers_shared.dylib", "libonnxruntime.dylib", false},
		{"libonnxruntime.so", "libonnxruntime.dylib", false},
		{"libonnxruntime.1.23.0.so", "libonnxruntime.dylib", false},
		{"libother.dylib", "libonnxruntime.dylib", false},
	} {
		if got := isLibrary(tc.found, tc.name); got != tc.want {
			t.Errorf("isLibrary(%q, %q) = %v, want %v", tc.found, tc.name, got, tc.want)
		}
	}
}

func createZip(t *testing.T, dir string, entries map[string]string) string {
	t.Helper()
	at := filepath.Join(dir, "onnxruntime-win-x64-1.23.0.zip")
	file, err := os.Create(at)
	if err != nil {
		t.Fatal(err)
	}
	writing := zip.NewWriter(file)
	for name, body := range entries {
		out, err := writing.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := out.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writing.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return at
}

func TestALibraryIsTakenOutOfAZip(t *testing.T) {
	dir := isolate(t)
	archive := createZip(t, dir, map[string]string{
		"onnxruntime-win-x64-1.23.0/lib/onnxruntime_providers_shared.dll": "another library",
		"onnxruntime-win-x64-1.23.0/lib/onnxruntime.dll":                  "the library",
	})
	found := release{archive: archive, sum: sum(t, archive), library: computeSum("the library")}

	at, err := unpack(archive, "onnxruntime.dll", found)
	if err != nil {
		t.Fatal(err)
	}
	held, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if string(held) != "the library" {
		t.Errorf("what came out of the zip was %q", held)
	}
}

func TestAZipHoldingNoSuchLibrarySaysSo(t *testing.T) {
	dir := isolate(t)
	archive := createZip(t, dir, map[string]string{"readme.txt": "nothing"})
	err := fromZip(archive, "onnxruntime.dll", filepath.Join(dir, "onnxruntime.dll"))
	if err == nil || !strings.Contains(err.Error(), "holds no onnxruntime.dll") {
		t.Errorf("a zip holding no such library was refused as %v", err)
	}
}

// A library already in the cache with the published sum needs no archive.
func TestALibraryInTheCacheNeedsNoArchive(t *testing.T) {
	dir := isolate(t)
	at := filepath.Join(dir, "libonnxruntime.dylib")
	if err := os.WriteFile(at, []byte("the library"), 0o755); err != nil {
		t.Fatal(err)
	}
	found := release{archive: filepath.Join(dir, "gone.tgz"), sum: computeSum("x"), library: computeSum("the library")}

	got, err := unpack(found.archive, "libonnxruntime.dylib", found)
	if err != nil || got != at {
		t.Errorf("unpack = %q, %v; want %q", got, err, at)
	}
}

func TestAFileThatIsNotGzipIsRefusedAsAGzipHeader(t *testing.T) {
	dir := isolate(t)
	archive := filepath.Join(dir, "plain.tgz")
	if err := os.WriteFile(archive, []byte("not gzip at all"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := fromTgz(archive, "lib", filepath.Join(dir, "lib")); !errors.Is(err, gzip.ErrHeader) {
		t.Errorf("unpack a file that is not gzip = %v", err)
	}
}
