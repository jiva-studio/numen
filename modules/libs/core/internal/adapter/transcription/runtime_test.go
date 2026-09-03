package transcription

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// packed is an archive laid out the way a mac release is: the library carrying
// its version, the plain name beside it as a link, and another library named for
// what it adds.
func packed(t *testing.T, dir string) string {
	t.Helper()
	at := filepath.Join(dir, "onnxruntime-osx-arm64-1.23.0.tgz")
	file, err := os.Create(at)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

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
		{name: under + "libonnxruntime.1.23.0.dylib", kind: tar.TypeReg, body: "the library"},
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
	return at
}

// A mac library carries its version before its extension, and the plain name
// beside it is a link to it. What is taken out of the archive is the file.
func TestTheMacLibraryIsTakenOutOfTheArchive(t *testing.T) {
	dir := t.TempDir()
	at, err := unpacked(packed(t, dir), "libonnxruntime.dylib")
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
