package onnxruntime

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// packed is an archive laid out the way a mac release is: the library carrying
// its version, the plain name beside it as a link, and another library named for
// what it adds. What comes back is where it is and the sums it and the library
// inside it carry.
func packed(t *testing.T, dir string) published {
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
	return published{archive: at, sum: sum(t, at), library: summed(body)}
}

// sum is what one file on this machine carries.
func sum(t *testing.T, at string) string {
	t.Helper()
	held, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	return summed(string(held))
}

func summed(body string) string {
	held := sha256.Sum256([]byte(body))
	return hex.EncodeToString(held[:])
}

// A mac library carries its version before its extension, and the plain name
// beside it is a link to it. What is taken out of the archive is the file.
func TestTheMacLibraryIsTakenOutOfTheArchive(t *testing.T) {
	dir := t.TempDir()
	found := packed(t, dir)
	at, err := unpacked(found.archive, "libonnxruntime.dylib", found)
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
	found := packed(t, dir)
	found.sum = summed("another archive")

	at, err := unpacked(found.archive, "libonnxruntime.dylib", found)
	if err == nil {
		t.Fatalf("a library was taken out of it, at %s", at)
	}
	if _, err := os.Stat(filepath.Join(dir, "libonnxruntime.dylib")); err == nil {
		t.Error("a library was left behind")
	}
}

// A library dropped into the cache under the name a fetched one is kept under
// is not the library this release publishes, and is written over by the one
// that is.
func TestALibraryInTheCacheThatIsNotTheOnePublishedIsWrittenOver(t *testing.T) {
	dir := t.TempDir()
	found := packed(t, dir)
	at := filepath.Join(dir, "libonnxruntime.dylib")
	if err := os.WriteFile(at, []byte("something else"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := unpacked(found.archive, "libonnxruntime.dylib", found); err != nil {
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
