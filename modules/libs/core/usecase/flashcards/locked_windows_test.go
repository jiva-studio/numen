package flashcards

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A run file another program holds open is one run of the log that could not be
// acted on. Windows refuses the open for as long as the handle stands, and a
// synchroniser and a backup reader each hold a file for moments at a time.
func TestARunHeldOpenByAnotherProgramIsOneSkippedRun(t *testing.T) {
	store := answering{why: &os.PathError{
		Op: "open", Path: "run.jsonl", Err: windows.ERROR_SHARING_VIOLATION,
	}}

	ran, err := Log{}.Run(t.Context(), store, port.Entry{Name: "run.jsonl"})
	if err != nil {
		t.Fatalf("a file another program holds refused the whole history: %v", err)
	}
	if !ran.Shut || ran.Skipped != 1 || len(ran.Answers) != 0 {
		t.Errorf("the run came back as %+v", ran)
	}
}
