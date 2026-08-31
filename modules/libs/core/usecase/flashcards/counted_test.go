package flashcards_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// What a person answered on a day is counted from the vault's own answers, and
// the days are named for the day they began on.
func TestWhatWasAnsweredIsCountedByDay(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	other := history.CardFace{Card: "zpqrstvwxy", Face: "Recognise"}

	yesterday := time.Now().AddDate(0, 0, -1)
	past := s.run(t, yesterday)
	if _, err := past.Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := past.Answer(t.Context(), other, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Answered != 3 {
		t.Errorf("the vault holds %d answers, want 3", got.Answered)
	}
	if got.Days[today.Names(yesterday)].Answered != 2 {
		t.Errorf("yesterday came to %d, want 2: %v", got.Days[today.Names(yesterday)].Answered, got.Days)
	}
	if got.Days[today.Names(time.Now())].Answered != 1 {
		t.Errorf("today came to %d, want 1: %v", got.Days[today.Names(time.Now())].Answered, got.Days)
	}
	if got.Streak != 2 {
		t.Errorf("the streak is %d, want the two days answered", got.Streak)
	}
}

// A vault nobody has answered has no days and no streak, which is an answer and
// not a failure.
func TestAVaultNobodyAnsweredHasNoDays(t *testing.T) {
	s := opened(t, vault)

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Days) != 0 || got.Streak != 0 || got.Answered != 0 {
		t.Errorf("counted %+v", got)
	}
}

// A run that has not changed is not read again. What a day came to is a sum, so
// the counting is kept by run and only what grew is counted afresh — which is
// what a schedule cannot do, because where an answer leaves a card depends on
// every answer before it.
func TestARunThatHasNotChangedIsNotCountedAgain(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}

	first := s.run(t, time.Now().AddDate(0, 0, -1))
	if _, err := first.Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.counted.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}

	// The run is left where it stands and its content is replaced with a line
	// nothing can read. A count that read it again would lose the day it holds.
	held := runsOf(t, s)
	if len(held) != 1 {
		t.Fatalf("the vault holds %v", held)
	}
	torn := filepath.Join(s.vault.Path, ".numen", "flashcards", held[0])
	was, err := os.ReadFile(torn)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(torn, spaces(len(was)), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Answered != 1 {
		t.Errorf("a run of the same length was read again: %+v", got)
	}
}

// A run that grew is counted afresh, so a sitting's own answers are in the
// counting as they are written.
func TestARunThatGrewIsCountedAfresh(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	other := history.CardFace{Card: "zpqrstvwxy", Face: "Recognise"}

	record := s.run(t, time.Now())
	if _, err := record.Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	first, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if first.Answered != 1 {
		t.Fatalf("counted %+v", first)
	}

	if _, err := record.Answer(t.Context(), other, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	second, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if second.Answered != 2 {
		t.Errorf("the answer written after the counting is not in it: %+v", second)
	}
}

// One identifier is one answer, however many files carry it. A synchroniser
// that met a conflict leaves a second copy of a run beside the first, and the
// day holds what the person answered.
func TestARunCopiedUnderAnotherNameIsCountedOnce(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	conflicted(t, s)

	// Once from the files, and again from what the first counting kept.
	for _, from := range []string{"the files", "the counting kept"} {
		got, err := s.counted.Execute(t.Context(), s.vault)
		if err != nil {
			t.Fatal(err)
		}
		if got.Answered != 1 {
			t.Errorf("counted from %s, the vault holds %d answers, want the one given",
				from, got.Answered)
		}
		if day := got.Days[today.Names(time.Now())].Answered; day != 1 {
			t.Errorf("counted from %s, today came to %d, want 1: %v", from, day, got.Days)
		}
	}
}

// conflicted puts a copy of every run beside it, under the name a synchroniser
// that met a conflict leaves.
func conflicted(t *testing.T, s vaulted) {
	t.Helper()
	at := filepath.Join(s.vault.Path, ".numen", "flashcards")
	for _, name := range runsOf(t, s) {
		raw, err := os.ReadFile(filepath.Join(at, name))
		if err != nil {
			t.Fatal(err)
		}
		beside := strings.TrimSuffix(name, flashcards.Suffix) +
			" (conflicted copy)" + flashcards.Suffix
		if err := os.WriteFile(filepath.Join(at, beside), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// runsOf is the files the vault's answers folder holds.
func runsOf(t *testing.T, s vaulted) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(s.vault.Path, ".numen", "flashcards"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, one := range entries {
		out = append(out, one.Name())
	}
	return out
}

// spaces is a file of one length holding nothing that reads as an answer.
func spaces(n int) []byte {
	out := make([]byte, n)
	for at := range out {
		out[at] = ' '
	}
	return out
}

// What is still to come is counted by the day it falls on, so a person can see
// the week ahead of them as well as the year behind.
func TestWhatIsStillToComeIsCountedByDay(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}

	// Answered easily, so it is days away rather than minutes.
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, history.Easy, 0); err != nil {
		t.Fatal(err)
	}

	schedules, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	due := schedules[on].Due

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Due[today.Names(due)] != 1 {
		t.Errorf("the day it falls on comes to %d, want the card: %v", got.Due[today.Names(due)], got.Due)
	}
	if len(got.Due) != 1 {
		t.Errorf("what is still to come is %v, want the one card", got.Due)
	}
}

// A card nobody has answered is not still to come: what a person owes now is
// what the front door counts, and this says what is after it.
func TestACardNobodyAnsweredIsNotStillToCome(t *testing.T) {
	s := opened(t, vault)

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Due) != 0 {
		t.Errorf("a vault nobody answered has %v still to come", got.Due)
	}
}

// countedCache is the day counts as a test writes them: enough of the shape to
// say what the cache claims, and to say it under a version of its own.
type countedCache struct {
	V    int                `json:"v"`
	Runs []countedCachedRun `json:"runs"`
}

type countedCachedRun struct {
	Name string                   `json:"name"`
	Size int                      `json:"size"`
	Days map[string]history.Tally `json:"days"`
	IDs  []string                 `json:"ids"`
}

// claiming puts a cache of its own over the vault's counting: the runs are the
// ones a counting just wrote, so the cache names this vault's files at the
// length they stand at, and what each day came to is the test's to say.
func claiming(t *testing.T, s vaulted, version int, days map[string]history.Tally) {
	t.Helper()
	if _, err := s.counted.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}
	raw, err := s.counted.Kept.Read(t.Context(), s.vault.ID)
	if err != nil {
		t.Fatal(err)
	}
	var was countedCache
	if err := json.Unmarshal(raw, &was); err != nil {
		t.Fatal(err)
	}
	if len(was.Runs) != 1 {
		t.Fatalf("the counting was kept from %d runs, want the one", len(was.Runs))
	}
	was.V, was.Runs[0].Days = version, days

	now, err := json.Marshal(was)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.counted.Kept.Write(t.Context(), s.vault.ID, now); err != nil {
		t.Fatal(err)
	}
}

// A cache written by a build that kept the counting in another shape is not
// read: what it holds is not what this build would have written, and the
// answers are there to be counted again.
func TestACacheOfAnotherShapeIsCountedAfresh(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}

	claiming(t, s, 0, map[string]history.Tally{"1999-01-01": {Answered: 99, Good: 99}})

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if _, held := got.Days["1999-01-01"]; held {
		t.Errorf("a cache of another shape was believed: %v", got.Days)
	}
	if got.Answered != 1 {
		t.Errorf("the vault holds %d answers, want the one written", got.Answered)
	}
}

// The same cache under the shape this build writes is believed, which is what
// says the shape is what the reading turns on and not the file's name.
func TestACacheOfThisShapeIsBelieved(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}

	claiming(t, s, 2, map[string]history.Tally{"1999-01-01": {Answered: 99, Good: 99}})

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Days["1999-01-01"].Answered != 99 {
		t.Errorf("the cache was not read: %v", got.Days)
	}
}

// A build that keeps nothing counts the whole log at every launch, and says the
// same as one that keeps it.
func TestAVaultIsCountedWithNothingKept(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}

	kept := s.counted
	kept.Kept = nil

	got, err := kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Answered != 1 || got.Days[today.Names(time.Now())].Answered != 1 {
		t.Errorf("counted %+v", got)
	}
}

// What is still to come is worked out from the schedules. A counting with no
// scheduler behind it says what was answered and nothing about what is coming,
// rather than refusing to count at all.
func TestWithNoSchedulerNothingIsStillToCome(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, history.Easy, 0); err != nil {
		t.Fatal(err)
	}

	blind := s.counted
	blind.Schedules = flashcards.Schedules{Logs: s.logs}

	got, err := blind.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Due) != 0 {
		t.Errorf("what is still to come is %v, want nothing", got.Due)
	}
	if got.Answered != 1 {
		t.Errorf("the vault holds %d answers, want the one written", got.Answered)
	}
}

// A card owed today is not still to come either, however it was answered.
func TestACardOwedTodayIsNotStillToCome(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}

	// Answered again, so it comes back in minutes and is owed today.
	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, history.Again, 0); err != nil {
		t.Fatal(err)
	}

	got, err := s.counted.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Due) != 0 {
		t.Errorf("a card owed today is counted as still to come: %v", got.Due)
	}
}
