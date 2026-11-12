package chunking

import "testing"

// A large chunk containing only part of a small one yields a hit near its end
// with nothing after it. Where another large chunk contains the whole small
// chunk, that is the one shown to the user.
func TestASmallChunkGoesBelowTheLargeOneContainingItWhole(t *testing.T) {
	large := []Chunk{
		{Start: 0, Length: 2372},
		{Start: 1887, Length: 1417},
	}
	small := []Chunk{{Start: 1887, Length: 626}}

	target := enclose(large, small)
	if len(target[0].Small) != 0 {
		t.Errorf("it went under the chunk that cuts it off: %v", target[0].Small)
	}
	if len(target[1].Small) != 1 || target[1].Small[0].Start != 1887 {
		t.Errorf("it did not go under the chunk holding it whole: %v", target[1].Small)
	}
}

// Where no large chunk holds the whole of it, the middle decides, as it always
// did.
func TestASmallChunkNoLargeOneContainsWhole(t *testing.T) {
	large := []Chunk{
		{Start: 0, Length: 100},
		{Start: 60, Length: 100},
	}
	small := []Chunk{{Start: 50, Length: 80}}

	target := enclose(large, small)
	if len(target[0].Small) != 1 {
		t.Errorf("the middle at 90 did not put it under the first: %v", target)
	}
}

// A small chunk lying before the first large chunk belongs to none, and the
// ones after it still go under the large chunk holding them.
func TestASmallChunkBeforeEveryLargeOneDoesNotHideTheNext(t *testing.T) {
	large := []Chunk{{Start: 100, Length: 50}}
	small := []Chunk{{Start: 0, Length: 10}, {Start: 110, Length: 10}}

	target := enclose(large, small)
	if len(target[0].Small) != 1 || target[0].Small[0].Start != 110 {
		t.Errorf("the chunk at 110 did not go under the large chunk: %v", target[0].Small)
	}
}
