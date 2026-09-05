package review_test

import (
	"bytes"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// logSeeds are the shapes a log arrives in: an answer, an answer taken back, a
// run that stopped part way through a line, a line of a version nobody knows, a
// line missing what a line must carry, a blank line, an instant written with an
// offset, JSON that is not a line at all, and bytes that are no log at all.
var logSeeds = []string{
	`{"v":1,"id":"a","card":"0123456789","face":"Word","at":"2026-01-01T00:00:00.000Z","rating":3,"ms":1200}` + "\n",
	`{"v":1,"id":"b","at":"2026-01-01T00:00:01.000Z","undo":"a"}` + "\n",
	`{"v":1,"id":"a","card":"c","face":"f","at":"2026-01-01T00:00:00.000Z","rating":3}` + "\n" + `{"v":1,"id":"b","car`,
	`{"v":2,"id":"a","card":"c","face":"f","at":"2026-01-01T00:00:00.000Z","rating":3}` + "\n",
	`{"v":1,"id":"a","card":"c","at":"2026-01-01T00:00:00.000Z","rating":3}` + "\n",
	`{"v":1,"id":"a","card":"c","face":"f","at":"2026-01-01T00:00:00.000Z","rating":9}` + "\n",
	`{"v":1,"id":"a","card":"c","face":"f","at":"2026-01-01T03:00:00.000+03:00","rating":3}` + "\n",
	"\n\n\n",
	"[]\n",
	"\x00\xff\xfe\n",
	"",
}

// Nothing is dropped without being counted: every line a log holds is an answer
// read out of it or one it says it could not act on.
//
// A log is appended to by a process that may stop at any moment and is carried
// between machines by something that knows nothing about it, so the bytes are a
// stranger's.
func FuzzReadLog(f *testing.F) {
	for _, seed := range logSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		answers, skipped := review.Read([]byte(raw))
		if held := lines(raw); len(answers)+skipped != held {
			t.Fatalf("%q holds %d lines and was read as %d answers and %d skipped",
				raw, held, len(answers), skipped)
		}
		for _, a := range answers {
			if a.ID == "" {
				t.Fatalf("%q was read as an answer of no identifier", raw)
			}
			if a.TakesBack() {
				continue
			}
			if a.CardFace.Card == "" || a.CardFace.Face == "" || !a.Rating.Valid() {
				t.Fatalf("%q was read as %+v, which answers nothing", raw, a)
			}
		}
	})
}

// lines is how many lines of a log there are to account for: the ones carrying
// something, and the run after the last newline, which is a line that did not
// land whole whatever it holds.
func lines(raw string) int {
	cut := strings.LastIndex(raw, "\n")
	if cut < 0 {
		if strings.TrimSpace(raw) == "" {
			return 0
		}
		return 1
	}
	held := 0
	if cut+1 < len(raw) {
		held++
	}
	for _, line := range strings.Split(raw[:cut], "\n") {
		if strings.TrimSpace(line) != "" {
			held++
		}
	}
	return held
}

// An answer written to the log comes back as itself. The file is the history a
// schedule is replayed from, so what is written has to survive being read.
func FuzzWriteAnswer(f *testing.F) {
	f.Add("a", "0123456789", "Word", int64(1767225600000), uint8(3), int64(1200), "")
	f.Add("b", "", "", int64(0), uint8(0), int64(0), "a")
	f.Add("", "c", "f", int64(-62135596800000), uint8(1), int64(-1), "")
	f.Add("«»\x00", "карточка", "文字", int64(253402300799000), uint8(4), int64(1), "")

	f.Fuzz(func(t *testing.T, id, card, face string, ms int64, rating uint8, took int64, undoes string) {
		// A stamp is written as four digits of year and read back as RFC 3339,
		// so an instant outside the years a stamp can spell is not one a log
		// carries.
		at := time.UnixMilli(ms).UTC()
		if at.Year() < 1 || at.Year() > 9999 {
			return
		}
		// JSON is text, and a string that is not UTF-8 is written as the
		// replacement character and comes back as something else. A mark is
		// Crockford base32 and an identifier is minted here, so nothing this
		// application writes carries one; a face's name comes off a stencil's
		// heading and could.
		for _, said := range []string{id, card, face, undoes} {
			if !utf8.ValidString(said) {
				return
			}
		}

		want := review.Answer{ID: id, At: at, Undoes: undoes}
		if undoes == "" {
			want.CardFace = review.CardFaceID{Card: card, Face: face}
			want.Rating = review.Rating(rating)
			want.Took = time.Duration(took) * time.Millisecond
		}

		raw, err := review.Write(want)
		if err != nil {
			t.Fatalf("%+v could not be written: %v", want, err)
		}
		if n := bytes.Count(raw, []byte("\n")); n != 1 || raw[len(raw)-1] != '\n' {
			t.Fatalf("%+v was written as %q, which is not one line", want, raw)
		}

		answers, skipped := review.Read(raw)
		// A line the reader refuses is one it could not have written: it carries
		// no identifier, or it answers with a rating that is not one of the four.
		refused := want.ID == "" ||
			(!want.TakesBack() && (want.CardFace.Card == "" ||
				want.CardFace.Face == "" || !want.Rating.Valid()))
		if refused {
			if len(answers) != 0 || skipped != 1 {
				t.Fatalf("%+v was read back as %d answers and %d skipped",
					want, len(answers), skipped)
			}
			return
		}
		if len(answers) != 1 || skipped != 0 {
			t.Fatalf("%+v was written as %q and read back as %d answers and %d skipped",
				want, raw, len(answers), skipped)
		}
		if got := answers[0]; !got.At.Equal(want.At) ||
			got.ID != want.ID || got.CardFace != want.CardFace ||
			got.Rating != want.Rating || got.Took != want.Took ||
			got.Undoes != want.Undoes {
			t.Fatalf("%+v was written as %q and came back as %+v", want, raw, got)
		}
	})
}
