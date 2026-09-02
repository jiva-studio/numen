package container

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
)

// said is what the list of what is being done says about putting a transcript
// right.
func said(held *Transcribing, path string) (doing, failed string) {
	for _, at := range held.tasks.List() {
		if at.ID == correcting(path) {
			return at.Doing, at.Failed
		}
	}
	return "", ""
}

// An installation that did not ask for its transcripts to be put right asks
// nothing and says nothing.
func TestATranscriptIsPutRightOnlyWhereItWasAskedFor(t *testing.T) {
	held, v := listens(t, &deaf{}, "talks/one.mp3")
	held.cfg.SpeechProofreading = proofreading.Proofread{With: "nowhere"}

	held.correct(t.Context(), v, "talks/one.mp3", false)

	if doing, _ := said(held, "talks/one.mp3"); doing != "" {
		t.Errorf("a transcript nobody asked about is %q", doing)
	}
}

// A profile no settings name is a person's mistake, and they are shown it.
func TestAProfileNoSettingsNameIsShown(t *testing.T) {
	held, v := listens(t, &deaf{}, "talks/one.mp3")
	held.cfg.SpeechProofreading = proofreading.Proofread{With: "nowhere", Automatically: true}

	held.correct(t.Context(), v, "talks/one.mp3", false)

	doing, failed := said(held, "talks/one.mp3")
	if doing != "Proofreading a transcript" {
		t.Fatalf("the list says %q", doing)
	}
	if !strings.Contains(failed, "nowhere") {
		t.Errorf("the failure does not name the profile: %q", failed)
	}
}

// Silence carries no words, so nothing is asked about it.
func TestSilenceIsNotPutRight(t *testing.T) {
	by := &deaf{}
	held, v := listens(t, by, "talks/one.mp3")
	held.cfg.SpeechProofreading = proofreading.Proofread{With: "nowhere", Automatically: true}

	held.Start(v, "talks/one.mp3")
	held.Wait()

	if doing, failed := said(held, "talks/one.mp3"); doing != "" {
		t.Errorf("silence is %q, failing with %q", doing, failed)
	}
}
