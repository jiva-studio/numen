package flashcards

import (
	"context"
	"fmt"
	"os"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// refusing is a store whose files are all held open by somebody else.
type refusing struct{ port.DerivedStore }

func (refusing) Read(context.Context, string) ([]byte, error) {
	return nil, &os.PathError{Op: "open", Path: "run.jsonl", Err: windows.ERROR_SHARING_VIOLATION}
}

// A run file another program holds is one run of the log that could not be
// acted on. Windows refuses the open for as long as the handle stands, and the
// answers in every other file are still the person's.
func TestARunHeldOpenByAnotherProgramIsOneSkippedRun(t *testing.T) {
	ran, err := Log{}.Run(t.Context(), refusing{}, port.Stored{Name: "run.jsonl"})
	if err != nil {
		t.Fatalf("a file another program holds refused the whole history: %v", err)
	}
	if !ran.Shut || ran.Skipped != 1 || len(ran.Answers) != 0 {
		t.Errorf("the run came back as %+v", ran)
	}
}

// Anything else the store says is still trouble the caller is told about.
func TestARunThatWouldNotBeReadForAnyOtherReasonIsTrouble(t *testing.T) {
	if _, err := (Log{}).Run(t.Context(), saying{}, port.Stored{Name: "run.jsonl"}); err == nil {
		t.Error("a store that could not answer was taken as a run that was read")
	}
}

type saying struct{ port.DerivedStore }

func (saying) Read(context.Context, string) ([]byte, error) {
	return nil, fmt.Errorf("the disk is not there")
}
