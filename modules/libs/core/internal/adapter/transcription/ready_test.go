package transcription

import (
	"os"
	"path/filepath"
	"testing"
)

// writeEmptyFile writes an empty file under dir and answers with where it is.
func writeEmptyFile(t *testing.T, dir, name string) string {
	t.Helper()
	at := filepath.Join(dir, name)
	if err := os.WriteFile(at, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return at
}

// Ready is answered by what is on this machine: a path written down is the
// file it names, the four files of the transducer are looked for by name in the
// models folder, and a name that is neither is not fetched.
func TestReadyIsAnsweredByTheModelsOnThisMachine(t *testing.T) {
	dir := t.TempDir()
	runtime := writeEmptyFile(t, dir, "libonnxruntime.so")
	segmenter := writeEmptyFile(t, dir, "silero.onnx")
	for _, name := range []string{encoderFile, decoderFile, joinerFile, tokensFile} {
		writeEmptyFile(t, dir, name)
	}

	for name, c := range map[string]struct {
		change  func(*testing.T, *Config)
		isReady bool
	}{
		"every model is here": {func(*testing.T, *Config) {}, true},
		"the download switch changes nothing": {
			func(_ *testing.T, cfg *Config) { cfg.ShouldDownload = true }, true,
		},
		"a path written down names nothing": {
			func(_ *testing.T, cfg *Config) { cfg.Segmenter.Path = filepath.Join(dir, "gone.onnx") }, false,
		},
		"one of the four is written down and gone": {
			func(_ *testing.T, cfg *Config) { cfg.Model.Joiner = filepath.Join(dir, "gone.onnx") }, false,
		},
		"one of the four is not in the models folder": {
			func(t *testing.T, _ *Config) {
				if err := os.Remove(filepath.Join(dir, tokensFile)); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { writeEmptyFile(t, dir, tokensFile) })
			}, false,
		},
		"a name is nowhere to fetch from": {
			func(_ *testing.T, cfg *Config) { cfg.Segmenter.Path, cfg.Segmenter.Repo = "", "elsewhere.onnx" }, false,
		},
		"a model is not named at all": {
			func(_ *testing.T, cfg *Config) { cfg.Segmenter.Path = "" }, false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := Config{
				Runtime:   runtime,
				Dir:       dir,
				Model:     ParakeetModel{Repo: "https://example.invalid/parakeet/"},
				Segmenter: SegmenterModel{Path: segmenter},
			}
			c.change(t, &cfg)
			if got := Ready(cfg); got != c.isReady {
				t.Errorf("ready: %v", got)
			}
		})
	}
}
