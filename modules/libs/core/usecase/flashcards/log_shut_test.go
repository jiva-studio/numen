package flashcards

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// answering is a store that hands back what it was given for every name.
type answering struct {
	port.DerivedStore
	why error
}

func (a answering) Read(context.Context, string) ([]byte, error) { return nil, a.why }

// A run file nobody may open is one run of the log that could not be acted on.
// The answers in every other file are still the person's, so a schedule is
// worked out from them and says how much it did not see.
func TestARunNobodyMayOpenIsOneSkippedRun(t *testing.T) {
	store := answering{why: &os.PathError{Op: "open", Path: "run.jsonl", Err: fs.ErrPermission}}

	ran, err := Log{}.Run(t.Context(), store, port.Entry{Name: "run.jsonl"})
	if err != nil {
		t.Fatalf("a file nobody may open refused the whole history: %v", err)
	}
	if !ran.Shut || ran.Skipped != 1 || len(ran.Answers) != 0 {
		t.Errorf("the run came back as %+v", ran)
	}
}

// A file another machine's synchroniser took away between the listing and the
// reading is no run at all, and nothing was skipped by it.
func TestARunTakenAwayBeforeItWasReadIsNoRun(t *testing.T) {
	store := answering{why: &os.PathError{Op: "open", Path: "run.jsonl", Err: fs.ErrNotExist}}

	ran, err := Log{}.Run(t.Context(), store, port.Entry{Name: "run.jsonl"})
	if err != nil {
		t.Fatalf("a file that was gone refused the whole history: %v", err)
	}
	if !ran.Gone || ran.Skipped != 0 {
		t.Errorf("the run came back as %+v", ran)
	}
}

// Anything else the store says is trouble the caller is told about.
func TestARunThatWouldNotBeReadForAnyOtherReasonIsTrouble(t *testing.T) {
	store := answering{why: fmt.Errorf("the disk is not there")}

	if _, err := (Log{}).Run(t.Context(), store, port.Entry{Name: "run.jsonl"}); err == nil {
		t.Error("a store that could not answer was taken as a run that was read")
	}
}
