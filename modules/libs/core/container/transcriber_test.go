package container

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

// A transcript is put right at the profile the settings name for speech, and
// what puts it right is told it is proofreading speech. The profile named for a
// scanned reading is another setting and is not read here.
func TestATranscriptIsPutRightWithTheProfileNamedForSpeech(t *testing.T) {
	cfg := Config{ServiceDir: ".numen"}
	cfg.Proofreading = proofreading.Defaults()
	cfg.Proofreading.Profiles = map[string]proofreading.Profile{
		"speech": {Use: proofreading.UseAgent, Model: "one model", BatchSize: 1, InFlight: 1},
		"scans":  {Use: proofreading.UseAgent, Model: "another model", BatchSize: 1, InFlight: 1},
	}
	cfg.TranscriptProofreading = proofreading.Proofread{Profile: "speech"}
	cfg.ScanProofreading = proofreading.Proofread{Profile: "scans", IsAutomatic: true}

	var opened ProofreaderSpec
	cfg.AgentProofreader = func(said ProofreaderSpec) (port.Proofreader, error) {
		opened = said
		return nil, errors.New("nothing on this machine puts a transcript right")
	}

	held := cfg.OpenTranscriptionWorker(t.Context(), nil, run.NewTasks(), nil)
	_, _ = held.Proofread(t.Context(), domain.Vault{ID: "v", Path: t.TempDir()}, "talks/one.mp3")
	held.Wait()

	if opened.Model != "one model" {
		t.Errorf("a transcript was put right at the profile using %q", opened.Model)
	}
	if opened.Instruction != proofread.SpeechInstruction {
		t.Error("what puts a transcript right was not told it is proofreading speech")
	}
}

// An installation that names something to put a transcript right with offers
// the run, and one that names nothing does not.
func TestAProofreadingIsOfferedWhereAProfileIsNamed(t *testing.T) {
	cfg := Config{ServiceDir: ".numen"}
	if cfg.OpenTranscriptionWorker(t.Context(), nil, nil, nil).ProofreaderReady() {
		t.Error("an installation naming no profile offers the run")
	}

	cfg.TranscriptProofreading = proofreading.Proofread{Profile: "by hand"}
	if !cfg.OpenTranscriptionWorker(t.Context(), nil, nil, nil).ProofreaderReady() {
		t.Error("an installation naming a profile does not offer the run")
	}
}

// Models stay loaded only where the available memory covers them with headroom.
func TestModelsStayLoadedOnlyWhereMemoryCoversThemWithHeadroom(t *testing.T) {
	const models = 1000
	whole := int64(models * (100 + memoryHeadroomPercent) / 100)
	for name, c := range map[string]struct {
		available, models int64
		want              bool
	}{
		"memory covers the models and the headroom": {whole, models, true},
		"memory falls short by a byte":              {whole - 1, models, false},
		"memory is unknown":                         {0, models, false},
		"the size of the models is unknown":         {whole, 0, false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := hasEnoughMemory(c.available, c.models); got != c.want {
				t.Errorf("canHoldResident = %v", got)
			}
		})
	}
}

// The memory a model needs is counted in whole megabytes of what is available,
// with the headroom on top, so the answer follows the size of the files.
func TestTheTranscriberStaysLoadedOnlyWhereTheFilesFitInTheMemoryAvailable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	availableMB := settings.DetectHardware().AvailableRAMMB
	if availableMB <= 0 {
		if (Config{}).HasMemoryForTranscriber() {
			t.Error("models stay loaded where the memory available is not known")
		}
		return
	}
	available := availableMB * (1 << 20)

	for name, c := range map[string]struct {
		share float64
		want  bool
	}{
		"files of under half of what is free stay loaded": {0.4, true},
		"files of over half of what is free do not":       {0.6, false},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := Config{}
			cfg.Transcription.Model.Encoder = filepath.Join(dir, "encoder.onnx")
			cfg.Transcription.Model.Decoder = filepath.Join(dir, "decoder.onnx")
			cfg.Transcription.Model.Joiner = filepath.Join(dir, "joiner.onnx")
			cfg.Transcription.Model.Tokens = filepath.Join(dir, "tokens.txt")
			cfg.Transcription.Segmenter.Path = filepath.Join(dir, "segmenter.onnx")
			for _, file := range []string{
				cfg.Transcription.Model.Decoder, cfg.Transcription.Model.Joiner,
				cfg.Transcription.Model.Tokens, cfg.Transcription.Segmenter.Path,
			} {
				if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			// A sparse file takes the size without taking the room.
			big, err := os.Create(cfg.Transcription.Model.Encoder)
			if err != nil {
				t.Fatal(err)
			}
			if err := big.Truncate(int64(float64(available) * c.share)); err != nil {
				t.Fatal(err)
			}
			if err := big.Close(); err != nil {
				t.Fatal(err)
			}

			if got := cfg.HasMemoryForTranscriber(); got != c.want {
				t.Errorf("CanKeepTranscriberLoaded = %v, want %v", got, c.want)
			}
		})
	}
}

// Headroom is the share of the model files again, counted in whole hundredths
// of them.
func TestTheHeadroomIsAShareOfTheFilesCountedInHundredths(t *testing.T) {
	for name, c := range map[string]struct {
		available, models int64
		want              bool
	}{
		"twice the files is enough":                        {2000, 1000, true},
		"a byte short of twice the files is not":           {1999, 1000, false},
		"hundredths of the files are whole ones":           {1800, 990, true},
		"a byte short of those hundredths is not":          {1799, 990, false},
		"unknown memory holds even the smallest files out": {0, 99, false},
		"one byte of memory holds files under a hundredth": {1, 99, true},
		"a single byte of file is a size":                  {1 << 20, 1, true},
	} {
		t.Run(name, func(t *testing.T) {
			if got := hasEnoughMemory(c.available, c.models); got != c.want {
				t.Errorf("canHoldResident(%d, %d) = %v", c.available, c.models, got)
			}
		})
	}
}

// makeTranscriptionFiles lays down every file a transcription reads, and
// the configuration pointing at them.
func makeTranscriptionFiles(t *testing.T) Config {
	t.Helper()
	dir := t.TempDir()
	cfg := Config{}
	files := []*string{
		&cfg.Transcription.Runtime,
		&cfg.Transcription.Model.Encoder, &cfg.Transcription.Model.Decoder,
		&cfg.Transcription.Model.Joiner, &cfg.Transcription.Model.Tokens,
		&cfg.Transcription.Segmenter.Path,
	}
	for i, file := range files {
		*file = filepath.Join(dir, "file"+string(rune('a'+i)))
		if err := os.WriteFile(*file, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return cfg
}

func TestATranscriberIsReadyOnlyWhereEverythingItReadsIsHere(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg := makeTranscriptionFiles(t)
	if !cfg.TranscriberReady() {
		t.Error("everything is here and the transcriber is not ready")
	}
	if err := os.Remove(cfg.Transcription.Model.Joiner); err != nil {
		t.Fatal(err)
	}
	if cfg.TranscriberReady() {
		t.Error("a model is missing and the transcriber is ready")
	}
}

func TestATranscriberThatCannotBeOpenedGivesTheReasonAndNothingElse(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg := Config{}
	cfg.Transcription.Runtime = filepath.Join(t.TempDir(), "missing")
	transcriber, closer, why := cfg.Transcriber(t.Context())
	if why == nil {
		t.Fatal("no reason")
	}
	if transcriber != nil || closer != nil {
		t.Errorf("got a transcriber %v and a closer", transcriber)
	}
}

// A transcription is made with the transcriber it is given.
func TestATranscriptionUsesTheTranscriberItIsGiven(t *testing.T) {
	by := &struct{ port.Transcriber }{}

	if got := (Config{}).Transcribe(nil, by).By(); got != port.Transcriber(by) {
		t.Errorf("the transcriber is %v", got)
	}
}
