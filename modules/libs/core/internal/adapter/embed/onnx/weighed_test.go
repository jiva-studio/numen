package onnx

import (
	"os"
	"path/filepath"
	"testing"
)

// A cache files one copy of a model under a name nobody asked for and hangs the
// published names off it. Weighed by the names alone, a model already here comes
// to a few hundred bytes and the count stands at nothing for as long as the wait
// lasts.
func TestANameIsWeighedAsWhatItPointsAt(t *testing.T) {
	dir := t.TempDir()
	blobs := filepath.Join(dir, "blobs")
	snapshot := filepath.Join(dir, "snapshots", "abc", "onnx")
	for _, at := range []string{blobs, snapshot} {
		if err := os.MkdirAll(at, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	blob := filepath.Join(blobs, "0123456789abcdef")
	if err := os.WriteFile(blob, make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(blob, filepath.Join(snapshot, "model.onnx")); err != nil {
		t.Fatal(err)
	}

	files := []string{"model.onnx"}
	sizes := map[string]int64{"model.onnx": 4096}

	if got := weighed(dir, files, sizes); got != 4096 {
		t.Errorf("weighed = %d, want the size of the file the name points at", got)
	}
}

// A name pointing at nothing is a name whose file is not here.
func TestANameLeadingNowhereWeighsNothing(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(dir, "model.onnx")); err != nil {
		t.Fatal(err)
	}

	if got := weighed(dir, []string{"model.onnx"}, map[string]int64{"model.onnx": 4096}); got != 0 {
		t.Errorf("weighed = %d, want nothing", got)
	}
}

// A file still coming down counts for what has arrived of it.
func TestAFilePartWrittenCountsForWhatIsThere(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "model.onnx.incomplete"), make([]byte, 1000), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := weighed(dir, []string{"model.onnx"}, map[string]int64{"model.onnx": 4096}); got != 1000 {
		t.Errorf("weighed = %d, want what has arrived", got)
	}
}
