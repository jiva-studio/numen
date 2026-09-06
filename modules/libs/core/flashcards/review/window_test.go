package review

import (
	"testing"
	"time"
)

// The days a card may be moved between are the interval either side of itself,
// by the slack its length carries. The window is named in whole days counted
// from the answer, and an interval outside the ends opens none at all.
func TestTheWindowACardMayBeMovedInside(t *testing.T) {
	for _, one := range []struct {
		days        float64
		first, last int
		opens       bool
	}{
		// Short of the first interval a card may be moved within, and past the
		// last.
		{days: 2},
		{days: 91},
		// Two days either side of a week, and three either side of twenty days.
		{days: 5, first: 4, last: 6, opens: true},
		{days: 7, first: 5, last: 9, opens: true},
		{days: 20, first: 17, last: 23, opens: true},
		{days: 90, first: 84, last: 96, opens: true},
	} {
		away := time.Duration(one.days * 24 * float64(time.Hour))
		first, last, opens := window(away)
		if opens != one.opens {
			t.Errorf("an interval of %g days opens a window: %t, want %t",
				one.days, opens, one.opens)
			continue
		}
		if !opens {
			continue
		}
		if first != one.first || last != one.last {
			t.Errorf("an interval of %g days may be put on days %d to %d, want %d to %d",
				one.days, first, last, one.first, one.last)
		}
	}
}
