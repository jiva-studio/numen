package review

import (
	"testing"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

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
		{"how far ahead a card may go", func(p *fsrs.Parameters) { p.MaximumInterval += 1 }},
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

	// Every weight, one at a time: a weight left out of the name is a schedule
	// worked out on other arithmetic and never recomputed.
	for i := range own().W {
		other := own()
		other.W[i] += 0.001
		if got := hashParameters(other); got == was {
			t.Errorf("weight %d changed and the name is still %q", i, got)
		}
	}

	// The same parameters are the same name, at this launch and the next.
	if got := hashParameters(own()); got != was {
		t.Errorf("the same parameters are named %q and then %q", was, got)
	}
}

// A field the arithmetic never reads decides no day, so moving it must not
// throw away every schedule in every vault. This stands for the upstream change
// that renames, reorders or adds a field: the name is worked out from the
// numbers by name, and not from how the library's struct happens to print.
func TestAFieldTheArithmeticDoesNotReadIsTheSameName(t *testing.T) {
	was := hashParameters(own())

	for _, one := range []struct {
		what   string
		change func(p *fsrs.Parameters)
	}{
		{"the fuzz", func(p *fsrs.Parameters) { p.EnableFuzz = true }},
		{"the library's own short-term scheduling", func(p *fsrs.Parameters) { p.EnableShortTerm = false }},
	} {
		t.Run(one.what, func(t *testing.T) {
			other := own()
			one.change(&other)
			if got := hashParameters(other); got != was {
				t.Errorf("%s changed and the name moved from %q to %q", one.what, was, got)
			}
		})
	}
}
