package source

import (
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// joins is a reply putting a run of lines together as one. A run stands within
// one batch, so a transcript put right this way is asked in whole batches.
func joins(at, through int, said string) string {
	return fmt.Sprintf("%d-%d|%s", at, through, said)
}

// wholly is a run over a transcript asked about in one batch.
func wholly(t *testing.T, says map[int]string, words ...string) (PutRight, domain.Vault, *shelf, string) {
	t.Helper()
	u, v, kept, _, hash := hearing(t, says, words...)
	u.Lines = len(words)
	return u, v, kept, hash
}

// A sentence the recording broke across two stretches comes back as one line,
// running from the first moment to the last.
func TestASentenceBrokenAcrossStretchesBecomesOneLine(t *testing.T) {
	words := []string{"Krishna is Raj. Krishna is", "connected with Raj Dila.", "Sure."}
	u, v, shelved, hash := wholly(t,
		map[int]string{0: joins(0, 1, "Krishna is Radha, Krishna is connected with Radhika.")},
		words...)

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	cues := cued(t, shelved, text.Said(text.ASR, hash))
	if len(cues) != 2 {
		t.Fatalf("the transcript says %+v", cues)
	}
	if cues[0].Text != "Krishna is Radha, Krishna is connected with Radhika." {
		t.Errorf("the first line says %q", cues[0].Text)
	}
	// The sentence runs from where the first stretch began to where the last
	// one ended.
	if cues[0].From != stretch(0).From || cues[0].To != stretch(1).To {
		t.Errorf("the sentence runs %d-%d", cues[0].From, cues[0].To)
	}
	if cues[1].Text != "Sure." || cues[1].From != stretch(2).From {
		t.Errorf("the line after it says %q at %d", cues[1].Text, cues[1].From)
	}
}

// What was heard is not written over by putting its lines together.
func TestJoiningLeavesWhatWasHeardWhereItIs(t *testing.T) {
	words := []string{"Krishna is Raj. Krishna is", "connected with Raj Dila."}
	u, v, shelved, hash := wholly(t,
		map[int]string{0: joins(0, 1, "Krishna is Radha, Krishna is connected with Radhika.")},
		words...)

	if _, err := u.Execute(t.Context(), v, recordingPath); err != nil {
		t.Fatal(err)
	}

	heard := cued(t, shelved, text.Artifact(text.ASR, hash))
	if len(heard) != 2 || heard[0].Text != words[0] || heard[1].Text != words[1] {
		t.Errorf("what was heard now says %+v", heard)
	}
}
