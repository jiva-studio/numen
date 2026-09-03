package flashcards_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

// jumbled is a history as a synchroniser leaves it: out of order, with a
// duplicated run and an answer taken back.
func jumbled() []flashcards.Answer {
	at := time.Date(2026, 8, 29, 9, 0, 0, 0, time.UTC)
	one := flashcards.Answer{
		ID: "b", CardFace: flashcards.CardFaceID{Card: "k7m2xq9fzp", Face: "Say it"},
		At: at.Add(time.Minute), Rating: flashcards.Good, Took: 4 * time.Second,
	}
	two := flashcards.Answer{
		ID: "a", CardFace: flashcards.CardFaceID{Card: "k7m2xq9fzp", Face: "Say it"},
		At: at, Rating: flashcards.Again, Took: 9 * time.Second,
	}
	three := flashcards.Answer{
		ID: "c", CardFace: flashcards.CardFaceID{Card: "zpqrstvwxy", Face: "Say it"},
		At: at.Add(2 * time.Minute), Rating: flashcards.Hard, Took: 6 * time.Second,
	}
	back := flashcards.Answer{
		ID: "d", Undoes: "c", At: at.Add(3 * time.Minute),
	}
	return []flashcards.Answer{one, two, three, back, one}
}

// A history put in order once is the history each of the four questions was
// putting in order for itself.
func TestGivingAHistoryOnceAnswersWhatGivingItFourTimesAnswered(t *testing.T) {
	answers := jumbled()
	day := flashcards.Day{Starts: flashcards.DayStarts, In: time.UTC}
	by := flashcards.NewFSRS()
	named := day.Names(answers[0].At)
	under := map[flashcards.CardFaceID]string{
		{Card: "k7m2xq9fzp", Face: "Say it"}: "Sanskrit.md",
		{Card: "zpqrstvwxy", Face: "Say it"}: "Sanskrit.md",
	}

	given := flashcards.Give(answers)
	if len(given) != 2 {
		t.Fatalf("the history came to %d answers, want 2", len(given))
	}

	for face, want := range flashcards.Replay(day, by, answers) {
		if got := given.Replay(day, flashcards.By(by))[face]; got != want {
			t.Errorf("%+v stands at %+v, want %+v", face, got, want)
		}
	}
	for day1, want := range flashcards.Retained(by, day, answers) {
		if got := given.Retained(by, day)[day1]; got != want {
			t.Errorf("%s kept %+v, want %+v", day1, got, want)
		}
	}
	for path, want := range flashcards.Sat(day, named, answers, under, nil) {
		if got := given.Sat(day, named, under, nil)[path]; got != want {
			t.Errorf("%s spent %+v, want %+v", path, got, want)
		}
	}
	for face, want := range flashcards.Faced(day, named, answers) {
		if got := given.Faced(day, named)[face]; got != want {
			t.Errorf("%+v was faced %v, want %v", face, got, want)
		}
	}
}
