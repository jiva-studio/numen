package recognition

import (
	"os"
	"strings"
	"testing"
)

// A decoder reads a class as the entry one before it. The dictionary carries
// every class but the blank, and the last of them is the space.
func TestTheDictionaryHoldsAnEntryForEveryClassButTheBlank(t *testing.T) {
	cfg := Defaults()
	cfg.ShouldDownload = false
	_, found, err := locate(t.Context(), cfg)
	if err != nil {
		t.Skipf("no models on this machine: %v", err)
	}

	classes, dict, err := alphabet(found.recognise, cfg.Recognise)
	if err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(dict)
	if err != nil {
		t.Fatal(err)
	}
	held := strings.Split(strings.TrimSuffix(string(written), "\n"), "\n")

	if int64(len(held)) != classes-1 {
		t.Fatalf("%d classes are read out of %d entries, so the last %d write a question mark",
			classes, len(held), classes-1-int64(len(held)))
	}
	if last := held[len(held)-1]; last != " " {
		t.Errorf("the class a space is read as is %q", last)
	}
}
