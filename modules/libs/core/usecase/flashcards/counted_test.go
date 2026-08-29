package flashcards_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
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
	if got.Days[today.Names(yesterday)] != 2 {
		t.Errorf("yesterday came to %d, want 2: %v", got.Days[today.Names(yesterday)], got.Days)
	}
	if got.Days[today.Names(time.Now())] != 1 {
		t.Errorf("today came to %d, want 1: %v", got.Days[today.Names(time.Now())], got.Days)
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
