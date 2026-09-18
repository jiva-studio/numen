package review_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// An answer written down and read back is the answer that was given.
func TestAnAnswerComesBackAsItWasWritten(t *testing.T) {
	given := review.Answer{
		ID:       "01K3ZQ7X2M9QRSTVWXYZ012345",
		CardFace: review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"},
		At:       parseTime("2026-08-29T09:12:33.412Z"),
		Rating:   review.Good,
		Took:     4210 * time.Millisecond,
	}

	raw, err := review.Write(given)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Errorf("a line ends with a newline, and this one is %q", raw)
	}

	back, skipped := review.Read(raw)
	if skipped != 0 {
		t.Errorf("skipped %d of one good line", skipped)
	}
	if len(back) != 1 || back[0] != given {
		t.Errorf("read back %+v, want %+v", back, given)
	}
}

// An answer taking another back names it and carries no card of its own.
func TestAnAnswerTakenBackNamesTheOneItTakesBack(t *testing.T) {
	raw, err := review.Write(review.Answer{
		ID:     "01K3ZQ7X8B0CDEFGHJKMNPQRST",
		At:     parseTime("2026-08-29T09:12:41.006Z"),
		Undoes: "01K3ZQ7X2M9QRSTVWXYZ012345",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"card"`) || strings.Contains(string(raw), `"rating"`) {
		t.Errorf("a line taking an answer back carries no card: %s", raw)
	}

	back, _ := review.Read(raw)
	if len(back) != 1 || !back[0].IsUndo() || back[0].Undoes != "01K3ZQ7X2M9QRSTVWXYZ012345" {
		t.Errorf("read back %+v", back)
	}
}

// A run that stopped partway leaves a line that did not land whole. What
// follows the last newline is left out, and everything before it is read.
func TestALineThatDidNotLandWholeIsLeftOut(t *testing.T) {
	whole, err := review.Write(review.Answer{
		ID:       "01K3ZQ7X2M9QRSTVWXYZ012345",
		CardFace: review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"},
		At:       parseTime("2026-08-29T09:12:33.412Z"),
		Rating:   review.Good,
	})
	if err != nil {
		t.Fatal(err)
	}
	torn := append(append([]byte(nil), whole...), []byte(`{"v":1,"id":"01K3ZQ7`)...)

	back, skipped := review.Read(torn)
	if len(back) != 1 {
		t.Errorf("read %d answers, want the one that landed", len(back))
	}
	if skipped != 1 {
		t.Errorf("counted %d lines it could not act on, want 1", skipped)
	}
}

// A line nobody can act on is left out and counted, and the rest of the file is
// read: a history is not refused because one line of it is unreadable.
func TestALineNobodyCanActOnIsCounted(t *testing.T) {
	good := `{"v":1,"id":"01K3ZQ7X2M9QRSTVWXYZ012345","card":"k7m2xq9fzp","face":"Recognise","at":"2026-08-29T09:12:33.412Z","rating":3}`
	for name, line := range map[string]string{
		"not json at all":         `this is not a line`,
		"a version to come":       `{"v":2,"id":"01K","card":"k7m2xq9fzp","face":"F","at":"2026-08-29T09:12:33.412Z","rating":3}`,
		"no identifier":           `{"v":1,"card":"k7m2xq9fzp","face":"F","at":"2026-08-29T09:12:33.412Z","rating":3}`,
		"a rating out of range":   `{"v":1,"id":"01K","card":"k7m2xq9fzp","face":"F","at":"2026-08-29T09:12:33.412Z","rating":9}`,
		"no face":                 `{"v":1,"id":"01K","card":"k7m2xq9fzp","at":"2026-08-29T09:12:33.412Z","rating":3}`,
		"an instant nobody wrote": `{"v":1,"id":"01K","card":"k7m2xq9fzp","face":"F","at":"the other day","rating":3}`,
	} {
		t.Run(name, func(t *testing.T) {
			back, skipped := review.Read([]byte(line + "\n" + good + "\n"))
			if len(back) != 1 {
				t.Errorf("read %d answers, want the one good line", len(back))
			}
			if skipped != 1 {
				t.Errorf("counted %d, want 1", skipped)
			}
		})
	}
}

// The file is text in a person's own folder and travels between machines, so an
// instant carrying an offset is read as the instant it is.
func TestAnInstantIsReadWithWhateverOffsetItCarries(t *testing.T) {
	line := `{"v":1,"id":"01K","card":"k7m2xq9fzp","face":"F",` +
		`"at":"2026-08-29T11:12:33.412+02:00","rating":3}`

	back, skipped := review.Read([]byte(line + "\n"))
	if len(back) != 1 || skipped != 0 {
		t.Fatalf("read %d answers and skipped %d", len(back), skipped)
	}
	if want := parseTime("2026-08-29T09:12:33.412Z"); !back[0].At.Equal(want) {
		t.Errorf("read %v, want the same instant as %v", back[0].At, want)
	}
}

// A file with nothing in it is a run that answered nothing, and blank lines are
// no line at all.
func TestAnEmptyLogReadsAsNothing(t *testing.T) {
	for name, raw := range map[string]string{
		"nothing":     "",
		"one newline": "\n",
		"blank lines": "\n\n\n",
		"spaces":      "   \n",
	} {
		t.Run(name, func(t *testing.T) {
			back, skipped := review.Read([]byte(raw))
			if len(back) != 0 || skipped != 0 {
				t.Errorf("read %d answers and skipped %d, want none of either", len(back), skipped)
			}
		})
	}
}
