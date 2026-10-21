package engine

import (
	"os"
	"path/filepath"
	"testing"
)

// The size of the models is the sum of the files on this machine, and nothing
// where one is missing.
func TestModelBytesAreTheSumOfTheFilesHere(t *testing.T) {
	dir := t.TempDir()
	runtime := writeEmptyFile(t, dir, "libonnxruntime.so")
	segmenter := writeEmptyFile(t, dir, "silero.onnx")
	var want int64
	for i, name := range []string{encoderFile, decoderFile, joinerFile, tokensFile} {
		at := writeEmptyFile(t, dir, name)
		size := 10 * (i + 1)
		if err := os.WriteFile(at, make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
		want += int64(size)
	}
	cfg := Config{Runtime: runtime, Dir: dir, Segmenter: SegmenterModel{Path: segmenter}}
	cfg.Model.Repo = "https://example.invalid/parakeet/"

	if got := GetModelBytes(cfg); got != want {
		t.Errorf("the models are %d bytes, want %d", got, want)
	}

	cfg.Segmenter.Path = filepath.Join(dir, "gone.onnx")
	if got := GetModelBytes(cfg); got != 0 {
		t.Errorf("with a model gone the size is %d", got)
	}
}
