package container

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

// A transcript is put right at the profile the settings name for speech, and
// what puts it right is told it is proofreading speech. The profile named for a
// scanned reading is another setting and is not read here.
func TestATranscriptIsPutRightWithTheProfileNamedForSpeech(t *testing.T) {
	cfg := Config{ServiceDir: ".numen"}
	cfg.Proofreading = DefaultProofreading()
	cfg.Proofreading.Profiles = map[string]Profile{
		"speech": {Use: UseAgent, Model: "one model", BatchSize: 1, InFlight: 1},
		"scans":  {Use: UseAgent, Model: "another model", BatchSize: 1, InFlight: 1},
	}
	cfg.TranscriptProofreading = Proofread{Profile: "speech"}
	cfg.ScanProofreading = Proofread{Profile: "scans", IsAutomatic: true}

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
}

// An installation that names something to put a transcript right with offers
// the run, and one that names nothing does not.
func TestAProofreadingIsOfferedWhereAProfileIsNamed(t *testing.T) {
	cfg := Config{ServiceDir: ".numen"}
	if cfg.OpenTranscriptionWorker(t.Context(), nil, nil, nil).ProofreaderReady() {
		t.Error("an installation naming no profile offers the run")
	}

	cfg.TranscriptProofreading = Proofread{Profile: "by hand"}
	if !cfg.OpenTranscriptionWorker(t.Context(), nil, nil, nil).ProofreaderReady() {
		t.Error("an installation naming a profile does not offer the run")
	}
}

func TestATranscriberIsReadyOnlyWherePluginIsAvailable(t *testing.T) {
	cfg := Config{}
	if cfg.TranscriberReady() {
		t.Error("transcriber is ready when no plugin host is configured")
	}

	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "transcription")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.transcription",
		"name": "Transcription",
		"version": "1.0.0",
		"capabilities": ["transcription"],
		"targets": ["desktop.editor"]
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg.PluginHost = NewPluginHost("desktop.editor", tmp)
	if !cfg.TranscriberReady() {
		t.Error("transcriber is not ready when transcription plugin is present")
	}
}

func TestATranscriberThatCannotBeOpenedGivesTheReasonAndNothingElse(t *testing.T) {
	cfg := Config{}
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
