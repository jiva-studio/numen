package review_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/review"
)

// makeJumbledHistory is a history as a synchroniser leaves it: out of order,
// with a duplicated run and an answer taken back.
func makeJumbledHistory() []review.Answer {
	at := time.Date(2026, 8, 29, 9, 0, 0, 0, time.UTC)
	one := review.Answer{
		ID: "b", CardFace: review.CardFaceID{Card: "k7m2xq9fzp", Face: "Say it"},
		At: at.Add(time.Minute), Rating: review.Good, Took: 4 * time.Second,
	}
	two := review.Answer{
		ID: "a", CardFace: review.CardFaceID{Card: "k7m2xq9fzp", Face: "Say it"},
		At: at, Rating: review.Again, Took: 9 * time.Second,
	}
	three := review.Answer{
		ID: "c", CardFace: review.CardFaceID{Card: "zpqrstvwxy", Face: "Say it"},
		At: at.Add(2 * time.Minute), Rating: review.Hard, Took: 6 * time.Second,
	}
	back := review.Answer{
		ID: "d", Undoes: "c", At: at.Add(3 * time.Minute),
	}
	return []review.Answer{one, two, three, back, one}
}

// A history put in order once is the history each of the four questions was
// putting in order for itself.
func TestGivingAHistoryOnceAnswersWhatGivingItFourTimesAnswered(t *testing.T) {
	answers := makeJumbledHistory()
	day := review.Day{Starts: review.DayStartTime, In: time.UTC}
	named := day.GetName(answers[0].At)
	under := map[review.CardFaceID]string{
		{Card: "k7m2xq9fzp", Face: "Say it"}: "Sanskrit.md",
		{Card: "zpqrstvwxy", Face: "Say it"}: "Sanskrit.md",
	}

	given := review.OrderAnswers(answers)
	if len(given) != 2 {
		t.Fatalf("the history came to %d answers, want 2", len(given))
	}

	for path, want := range review.GetSpentUnder(day, named, answers, under, nil) {
		if got := given.GetSpentUnder(day, named, under, nil)[path]; got != want {
			t.Errorf("%s spent %+v, want %+v", path, got, want)
		}
	}
	for face, want := range review.GetFaced(day, named, answers) {
		if got := given.GetFaced(day, named)[face]; got != want {
			t.Errorf("%+v was faced %v, want %v", face, got, want)
		}
	}
}
