package flashcards_test

import (
	"testing"
)

// A reading works its order out once, however many callers ask for it.
func TestAReadingIsPutInOrderOnce(t *testing.T) {
	t.Parallel()
	l := load(t, loadCards, loadDays, loadPerDay)

	held, err := l.logRead(t)
	if err != nil {
		t.Fatal(err)
	}
	first, again := held.History(), held.History()
	if len(first) != loadDays*loadPerDay {
		t.Fatalf("the reading came to %d answers where the vault holds %d",
			len(first), loadDays*loadPerDay)
	}
	if &first[0] != &again[0] {
		t.Error("the reading was put in order twice")
	}
}
