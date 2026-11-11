package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/settings"
)

func writeEmptyFile(t *testing.T, dir, name string) string {
	t.Helper()
	pos := filepath.Join(dir, name)
	if err := os.WriteFile(pos, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return pos
}

// Ready is answered by what is on this machine: a path written down is the
// file it names, the four files of the transducer are looked for by name in the
// models folder, and a name that is neither is not fetched.
func TestReady_ChecksLocalModelFiles(t *testing.T) {
	dir := t.TempDir()
	runtime := writeEmptyFile(t, dir, "libonnxruntime.so")
	segmenterPath := writeEmptyFile(t, dir, "silero.onnx")
	for _, name := range []string{encoderFile, decoderFile, joinerFile, tokensFile} {
		writeEmptyFile(t, dir, name)
	}

	for name, c := range map[string]struct {
		change  func(*testing.T, *settings.Config)
		isReady bool
	}{
		"every model is here": {func(*testing.T, *settings.Config) {}, true},
		"the download switch changes nothing": {
			func(_ *testing.T, cfg *settings.Config) { cfg.ShouldDownload = true }, true,
		},
		"a path written down names nothing": {
			func(_ *testing.T, cfg *settings.Config) { cfg.Segmenter.Path = filepath.Join(dir, "gone.onnx") }, false,
		},
		"the runtime is written down and gone": {
			func(_ *testing.T, cfg *settings.Config) { cfg.Runtime = filepath.Join(dir, "gone.so") }, false,
		},
		"the encoder is written down and gone": {
			func(_ *testing.T, cfg *settings.Config) { cfg.Model.Encoder = filepath.Join(dir, "gone.onnx") }, false,
		},
		"the decoder is written down and gone": {
			func(_ *testing.T, cfg *settings.Config) { cfg.Model.Decoder = filepath.Join(dir, "gone.onnx") }, false,
		},
		"one of the four is written down and gone": {
			func(_ *testing.T, cfg *settings.Config) { cfg.Model.Joiner = filepath.Join(dir, "gone.onnx") }, false,
		},
		"one of the four is not in the models folder": {
			func(t *testing.T, _ *settings.Config) {
				t.Helper()
				if err := os.Remove(filepath.Join(dir, tokensFile)); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { writeEmptyFile(t, dir, tokensFile) })
			}, false,
		},
		"a name is nowhere to fetch from": {
			func(_ *testing.T, cfg *settings.Config) {
				cfg.Segmenter.Path, cfg.Segmenter.Repo = "", "elsewhere.onnx"
			}, false,
		},
		"a model is not named at all": {
			func(_ *testing.T, cfg *settings.Config) { cfg.Segmenter.Path = "" }, false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := settings.Config{
				Runtime:   runtime,
				Dir:       dir,
				Model:     settings.ParakeetModel{Repo: "https://example.invalid/parakeet/"},
				Segmenter: settings.SegmenterModel{Path: segmenterPath},
			}
			c.change(t, &cfg)
			if got := Ready(NewRuntime(), cfg); got != c.isReady {
				t.Errorf("ready: %v", got)
			}
		})
	}
}
