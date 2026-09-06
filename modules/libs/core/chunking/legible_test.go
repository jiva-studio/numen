package chunking

import "testing"

// A negative threshold asks for none, for both of them. A corpus written in a
// script these fractions were not measured on is indexed whole.
func TestANegativeThresholdAsksForNone(t *testing.T) {
	rubbish := "|| $$ ?? %%% ### @@@ ~~~ ^^^"
	plain := "the quick brown fox jumps over the lazy dog"

	kept := Legibility{Alphabetic: -1, Dirty: -1}
	for _, text := range []string{rubbish, plain} {
		if !legible(text, kept) {
			t.Errorf("with no threshold, %q was refused", text)
		}
	}

	// The thresholds as they stand still refuse what they were written for.
	defaults := Legibility{}.resolve()
	if legible(rubbish, defaults) {
		t.Errorf("%q passed the thresholds", rubbish)
	}
	if !legible(plain, defaults) {
		t.Errorf("%q was refused by the thresholds", plain)
	}
}
