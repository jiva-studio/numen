package review

import (
	"testing"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

// The days a card may be moved between are the interval either side of itself,
// by the slack its length carries. The window is named in whole days counted
// from the answer, and an interval outside the ends opens none at all.
func TestTheWindowACardMayBeMovedInside(t *testing.T) {
	for _, one := range []struct {
		days        float64
		first, last int
		isOpening   bool
	}{
		// Short of the first interval a card may be moved within, and past the
		// last.
		{days: 2},
		{days: 91},
		// Two days either side of a week, and three either side of twenty days.
		{days: 5, first: 4, last: 6, isOpening: true},
		{days: 7, first: 5, last: 9, isOpening: true},
		{days: 20, first: 17, last: 23, isOpening: true},
		{days: 90, first: 84, last: 96, isOpening: true},
	} {
		away := time.Duration(one.days * 24 * float64(time.Hour))
		first, last, opens := window(away)
		if opens != one.isOpening {
			t.Errorf("an interval of %g days opens a window: %t, want %t",
				one.days, opens, one.isOpening)
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

// own is the parameters this package schedules on.
func own() fsrs.Parameters {
	p := fsrs.DefaultParam()
	p.EnableFuzz = false
	return p
}

// What a schedule is filed under carries the numbers it was worked out on. A
// build spacing cards by other numbers must not read this build's stability and
// difficulty as its own: they mean what those numbers say they mean, and one
// read as the other is a wrong day given confidently.
func TestANumberTheArithmeticReadsIsAnotherName(t *testing.T) {
	was := hashParameters(own())

	for _, one := range []struct {
		what   string
		change func(p *fsrs.Parameters)
	}{
		{"the retention asked for", func(p *fsrs.Parameters) { p.RequestRetention += 0.01 }},
		{"how far ahead a card may go", func(p *fsrs.Parameters) { p.MaximumInterval++ }},
		{"the forgetting curve's decay", func(p *fsrs.Parameters) { p.Decay -= 0.01 }},
		{"the forgetting curve's factor", func(p *fsrs.Parameters) { p.Factor += 0.01 }},
	} {
		t.Run(one.what, func(t *testing.T) {
			other := own()
			one.change(&other)
			if got := hashParameters(other); got == was {
				t.Errorf("%s changed and the name is still %q", one.what, got)
			}
		})
	}

	for _, one := range []struct {
		what   string
		change func(p *fsrs.Parameters)
	}{
		{"a weight", func(p *fsrs.Parameters) { p.W[3] += 0.01 }},
		{"another weight", func(p *fsrs.Parameters) { p.W[11] += 0.01 }},
	} {
		t.Run(one.what, func(t *testing.T) {
			other := own()
			one.change(&other)
			if got := hashParameters(other); got == was {
				t.Errorf("%s changed and the name is still %q", one.what, got)
			}
		})
	}
}
