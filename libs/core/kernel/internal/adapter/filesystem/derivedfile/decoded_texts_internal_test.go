package derivedfile

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func getSegmentsOf(word string) []domain.TextSegment {
	return []domain.TextSegment{{Text: word}}
}

func TestTheDecodedTextsDropTheLeastRecentlyReadFileOverTheBudget(t *testing.T) {
	one := getSegmentsSize(getSegmentsOf("river"))
	c := NewDecodedTexts(2 * one)
	c.put("a", getSegmentsOf("river"))
	c.put("b", getSegmentsOf("ocean"))
	if _, ok := c.get("a"); !ok {
		t.Fatal("a was dropped within the budget")
	}

	c.put("c", getSegmentsOf("delta"))

	if _, ok := c.get("b"); ok {
		t.Error("b, read least recently, was kept over the budget")
	}
	for _, hash := range []string{"a", "c"} {
		if got, ok := c.get(hash); !ok || len(got) != 1 {
			t.Errorf("%s was dropped: %v", hash, got)
		}
	}
}

func TestTheDecodedTextsKeepNoFileLargerThanTheBudget(t *testing.T) {
	c := NewDecodedTexts(getSegmentsSize(getSegmentsOf("river")))
	c.put("a", getSegmentsOf("river"))
	c.put("b", getSegmentsOf("a much longer text"))

	if _, ok := c.get("b"); ok {
		t.Error("a file larger than the budget was kept")
	}
	if _, ok := c.get("a"); !ok {
		t.Error("a file larger than the budget pushed out the one kept")
	}
}

func TestNoDecodedTextsKeepNothing(t *testing.T) {
	var c *DecodedTexts
	c.put("a", getSegmentsOf("river"))
	if _, ok := c.get("a"); ok {
		t.Error("nil decoded texts returned a file")
	}
}
