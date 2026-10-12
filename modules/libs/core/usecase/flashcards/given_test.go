package flashcards_test

import (
	"testing"
)

// A loaded log works its order out once, however many callers ask for it.
func TestALoadedLogIsPutInOrderOnce(t *testing.T) {
	t.Parallel()
	l := load(t, loadCards, loadDays, loadPerDay)

	held, err := l.logRead(t)
	if err != nil {
		t.Fatal(err)
	}
	first, again := held.GetHistory(), held.GetHistory()
	if len(first) != loadDays*loadPerDay {
		t.Fatalf("the log came to %d answers where the vault holds %d",
			len(first), loadDays*loadPerDay)
	}
	if &first[0] != &again[0] {
		t.Error("the log was put in order twice")
	}
}
