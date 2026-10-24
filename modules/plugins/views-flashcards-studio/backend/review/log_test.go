package review_test

import (
	"bytes"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/review"
)

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// An answer written down and read back is the answer that was given.
func TestAnAnswerReachesBackAsItWasSaved(t *testing.T) {
	provided := review.Answer{
		ID:       "01K3ZQ7X2M9QRSTVWXYZ012345",
		CardFace: review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"},
		At:       parseTime("2026-08-29T09:12:33.412Z"),
		Rating:   review.Good,
		Took:     4210 * time.Millisecond,
	}

	raw, err := review.Write(provided)
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
	if len(back) != 1 || back[0] != provided {
		t.Errorf("read back %+v, want %+v", back, provided)
	}
}

// An answer taking another back names it and carries no card of its own.
func TestAnAnswerTakenBackNamesTheOneItConsumesBack(t *testing.T) {
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
	divided := append(append([]byte(nil), whole...), []byte(`{"v":1,"id":"01K3ZQ7`)...)

	back, skipped := review.Read(divided)
	if len(back) != 1 {
		t.Errorf("read %d answers, want the one that landed", len(back))
	}
	if skipped != 1 {
		t.Errorf("counted %d lines it could not act on, want 1", skipped)
	}
}

// A line nobody can act on is left out and counted, and the rest of the file is
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

// Each of the four is said in the word a person uses for it, which is what a
// log line and anything read by a person carry.
func TestEachRatingIsSaidInItsOwnWord(t *testing.T) {
	for _, one := range []struct {
		r    review.Rating
		want string
	}{
		{review.Again, "again"},
		{review.Hard, "hard"},
		{review.Good, "good"},
		{review.Easy, "easy"},
		{0, "unknown"},
		{review.Easy + 1, "unknown"},
	} {
		if got := one.r.String(); got != one.want {
			t.Errorf("%d is said as %q, want %q", one.r, got, one.want)
		}
	}
}

// Only the four are ratings. A number out of a file that is not one of them is
// a line nobody can act on, and saying so is what leaves it out.
func TestOnlyTheFourAreRatings(t *testing.T) {
	for _, r := range []review.Rating{
		review.Again, review.Hard, review.Good, review.Easy,
	} {
		if !r.Valid() {
			t.Errorf("%s is not one of the four", r)
		}
	}
	for _, r := range []review.Rating{0, review.Easy + 1, 255} {
		if r.Valid() {
			t.Errorf("%d passed as one of the four", r)
		}
	}
}

// makeJumbledHistory is a history as a synchroniser leaves it: out of order,
// with a duplicated run and an answer taken back.
func makeJumbledHistory() []review.Answer {
	pos := time.Date(2026, 8, 29, 9, 0, 0, 0, time.UTC)
	one := review.Answer{
		ID: "b", CardFace: review.CardFaceID{Card: "k7m2xq9fzp", Face: "Say it"},
		At: pos.Add(time.Minute), Rating: review.Good, Took: 4 * time.Second,
	}
	two := review.Answer{
		ID: "a", CardFace: review.CardFaceID{Card: "k7m2xq9fzp", Face: "Say it"},
		At: pos, Rating: review.Again, Took: 9 * time.Second,
	}
	three := review.Answer{
		ID: "c", CardFace: review.CardFaceID{Card: "zpqrstvwxy", Face: "Say it"},
		At: pos.Add(2 * time.Minute), Rating: review.Hard, Took: 6 * time.Second,
	}
	back := review.Answer{
		ID: "d", Undoes: "c", At: pos.Add(3 * time.Minute),
	}
	return []review.Answer{one, two, three, back, one}
}

// A history put in order once is the history each of the four questions was
// putting in order for itself.
func TestGivingAHistoryOnceAnswersWhatGivingItFourTimesAnswered(t *testing.T) {
	answers := makeJumbledHistory()
	day := review.Day{Starts: review.DayStartTime, In: time.UTC}
	named := day.GetName(answers[0].At)
	underMap := map[review.CardFaceID]string{
		{Card: "k7m2xq9fzp", Face: "Say it"}: "Sanskrit.md",
		{Card: "zpqrstvwxy", Face: "Say it"}: "Sanskrit.md",
	}

	provided := review.OrderAnswers(answers)
	if len(provided) != 2 {
		t.Fatalf("the history came to %d answers, want 2", len(provided))
	}

	for path, want := range review.GetSpentUnder(day, named, answers, underMap, nil) {
		if got := provided.GetSpentUnder(day, named, underMap, nil)[path]; got != want {
			t.Errorf("%s spent %+v, want %+v", path, got, want)
		}
	}
	for face, want := range review.GetFaced(day, named, answers) {
		if got := provided.GetFaced(day, named)[face]; got != want {
			t.Errorf("%+v was faced %v, want %v", face, got, want)
		}
	}
}

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
		if target := lines(raw); len(answers)+skipped != target {
			t.Fatalf("%q holds %d lines and was read as %d answers and %d skipped",
				raw, target, len(answers), skipped)
		}
		for _, a := range answers {
			if a.ID == "" {
				t.Fatalf("%q was read as an answer of no identifier", raw)
			}
			if a.IsUndo() {
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
	lastNewline := strings.LastIndex(raw, "\n")
	if lastNewline < 0 {
		if strings.TrimSpace(raw) == "" {
			return 0
		}
		return 1
	}
	target := 0
	if lastNewline+1 < len(raw) {
		target++
	}
	for _, line := range strings.Split(raw[:lastNewline], "\n") {
		if strings.TrimSpace(line) != "" {
			target++
		}
	}
	return target
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
		pos := time.UnixMilli(ms).UTC()
		if pos.Year() < 1 || pos.Year() > 9999 {
			return
		}
		// JSON is text, and a string that is not UTF-8 is written as the
		// replacement character and comes back as something else. A mark is
		// Crockford base32 and an identifier is minted here, so nothing this
		// application writes carries one; a face's name comes off a stencil's
		// heading and could.
		for _, response := range []string{id, card, face, undoes} {
			if !utf8.ValidString(response) {
				return
			}
		}

		want := review.Answer{ID: id, At: pos, Undoes: undoes}
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
		shouldFail := want.ID == "" ||
			(!want.IsUndo() && (want.CardFace.Card == "" ||
				want.CardFace.Face == "" || !want.Rating.Valid()))
		if shouldFail {
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
