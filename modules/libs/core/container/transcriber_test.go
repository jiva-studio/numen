package container

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// A transcript is put right at the profile the settings name for speech, and
// what puts it right is told it is correcting speech. The profile named for a
// scanned reading is another setting and is not read here.
func TestATranscriptIsPutRightWithTheProfileNamedForSpeech(t *testing.T) {
	cfg := Config{ServiceDir: ".numen"}
	cfg.Proofreading = proofreading.Defaults()
	cfg.Proofreading.Profiles = map[string]proofreading.Profile{
		"speech": {Use: proofreading.UseAgent, Model: "one model", BatchSize: 1, InFlight: 1},
		"scans":  {Use: proofreading.UseAgent, Model: "another model", BatchSize: 1, InFlight: 1},
	}
	cfg.SpeechProofreading = proofreading.Proofread{With: "speech"}
	cfg.ScanProofreading = proofreading.Proofread{With: "scans", Automatically: true}

	var opened AgentProofreading
	cfg.AgentProofreader = func(said AgentProofreading) (port.Proofreader, error) {
		opened = said
		return nil, errors.New("nothing on this machine puts a transcript right")
	}

	held := cfg.Transcribing(t.Context(), nil, task.New())
	_, _ = held.Proofread(t.Context(), domain.Vault{ID: "v", Path: t.TempDir()}, "talks/one.mp3")
	held.Wait()

	if opened.Model != "one model" {
		t.Errorf("a transcript was put right at the profile using %q", opened.Model)
	}
	if opened.Instruction != proofread.SpeechInstruction {
		t.Error("what puts a transcript right was not told it is correcting speech")
	}
}

// An installation that names something to put a transcript right with offers
// the run, and one that names nothing does not.
func TestAProofreadingIsOfferedWhereAProfileIsNamed(t *testing.T) {
	cfg := Config{ServiceDir: ".numen"}
	if cfg.Transcribing(t.Context(), nil, nil).ProofreaderReady() {
		t.Error("an installation naming no profile offers the run")
	}

	cfg.SpeechProofreading = proofreading.Proofread{With: "by hand"}
	if !cfg.Transcribing(t.Context(), nil, nil).ProofreaderReady() {
		t.Error("an installation naming a profile does not offer the run")
	}
}
