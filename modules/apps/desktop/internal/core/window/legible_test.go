package window

import "testing"

// A negative threshold asks for none, for both of them. A corpus written in a
// script these fractions were not measured on is indexed whole.
func TestANegativeThresholdAsksForNone(t *testing.T) {
	rubbish := "|| $$ ?? %%% ### @@@ ~~~ ^^^"
	plain := "the quick brown fox jumps over the lazy dog"

	kept := Sizes{Alphabetic: -1, Dirty: -1}
	for _, text := range []string{rubbish, plain} {
		if !legible(text, kept) {
			t.Errorf("with no threshold, %q was refused", text)
		}
	}

	// The thresholds as they stand still refuse what they were written for.
	standing := Sizes{}.resolve()
	if legible(rubbish, standing) {
		t.Errorf("%q passed the thresholds", rubbish)
	}
	if !legible(plain, standing) {
		t.Errorf("%q was refused by the thresholds", plain)
	}
}
