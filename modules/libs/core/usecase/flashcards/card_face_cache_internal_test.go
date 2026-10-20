package flashcards

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func deckOf(n int) deckFaces { return deckFaces{faces: make([]CardFace, n)} }

func keyOf(path string) deckKey { return deckKey{vault: "v", path: path} }

func TestTheLeastRecentlyUsedDeckIsForgottenAloneWhenTheFacesPassTheBound(t *testing.T) {
	t.Parallel()
	c := newFaceCache()
	c.put(0, keyOf("a"), deckOf(2), 6)
	c.put(0, keyOf("b"), deckOf(2), 6)
	c.put(0, keyOf("c"), deckOf(2), 6)
	c.get(keyOf("a"))

	c.put(0, keyOf("d"), deckOf(2), 6)

	if c.hasDeck(keyOf("b")) {
		t.Error("the least recently used deck is still held")
	}
	for _, kept := range []string{"a", "c", "d"} {
		if !c.hasDeck(keyOf(kept)) {
			t.Errorf("deck %s was forgotten with it", kept)
		}
	}
	if c.getSize() != 6 {
		t.Errorf("%d faces held, want 6", c.getSize())
	}
}

func TestTheFacesHeldNeverPassTheBound(t *testing.T) {
	t.Parallel()
	c := newFaceCache()
	for i := range 50 {
		c.put(0, keyOf(string(rune('a'+i%26))+string(rune('a'+i/26))), deckOf(1+i%4), 10)
		if c.getSize() > 10 {
			t.Fatalf("%d faces held past a bound of 10", c.getSize())
		}
	}
}

func TestADeckHoldingMoreFacesThanTheBoundIsNotStored(t *testing.T) {
	t.Parallel()
	c := newFaceCache()
	c.put(0, keyOf("small"), deckOf(1), 3)
	c.put(0, keyOf("big"), deckOf(4), 3)

	if c.hasDeck(keyOf("big")) || !c.hasDeck(keyOf("small")) {
		t.Error("an oversize deck emptied the cache or was stored")
	}
}

func TestReplacingADeckCountsItsFacesOnce(t *testing.T) {
	t.Parallel()
	c := newFaceCache()
	c.put(0, keyOf("a"), deckOf(3), 10)
	c.put(0, keyOf("a"), deckOf(2), 10)

	if c.getSize() != 2 {
		t.Errorf("%d faces held, want 2", c.getSize())
	}
	c.invalidate(domain.VaultID("v"), []string{"a"})
	if c.getSize() != 0 {
		t.Errorf("%d faces held after forgetting the only deck", c.getSize())
	}
}
