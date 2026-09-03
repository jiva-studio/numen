package cutting

import "testing"

// A large chunk holding only part of a small one shows a hit near its end with
// nothing after it. Where another large chunk holds the whole of it, that is
// the one a person is shown.
func TestASmallChunkGoesUnderTheLargeOneHoldingItWhole(t *testing.T) {
	large := []Chunk{
		{Start: 0, Length: 2372},
		{Start: 1887, Length: 1417},
	}
	small := []Chunk{{Start: 1887, Length: 626}}

	held := enclose(large, small)
	if len(held[0].Small) != 0 {
		t.Errorf("it went under the chunk that cuts it off: %v", held[0].Small)
	}
	if len(held[1].Small) != 1 || held[1].Small[0].Start != 1887 {
		t.Errorf("it did not go under the chunk holding it whole: %v", held[1].Small)
	}
}

// Where no large chunk holds the whole of it, the middle decides, as it always
// did.
func TestASmallChunkNoLargeOneHoldsWhole(t *testing.T) {
	large := []Chunk{
		{Start: 0, Length: 100},
		{Start: 60, Length: 100},
	}
	small := []Chunk{{Start: 50, Length: 80}}

	held := enclose(large, small)
	if len(held[0].Small) != 1 {
		t.Errorf("the middle at 90 did not put it under the first: %v", held)
	}
}
