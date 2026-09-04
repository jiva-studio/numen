package review

import (
	"testing"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

// What a schedule is filed under carries the parameters it was worked out on. A
// build spacing cards by other weights must not read this build's stability and
// difficulty as its own: the numbers mean what the weights say they mean, and
// one read as the other is a wrong day given confidently.
func TestParametersThatDifferAtAllAreAnotherName(t *testing.T) {
	own := fsrs.DefaultParam()
	own.EnableFuzz = false
	was := weighed(own)

	for _, one := range []struct {
		what   string
		change func(p *fsrs.Parameters)
	}{
		{"a weight", func(p *fsrs.Parameters) { p.W[0] += 0.001 }},
		{"the retention asked for", func(p *fsrs.Parameters) { p.RequestRetention += 0.01 }},
		{"how far ahead a card may go", func(p *fsrs.Parameters) { p.MaximumInterval += 1 }},
		{"the fuzz", func(p *fsrs.Parameters) { p.EnableFuzz = true }},
	} {
		t.Run(one.what, func(t *testing.T) {
			other := own
			one.change(&other)
			if got := weighed(other); got == was {
				t.Errorf("%s changed and the name is still %q", one.what, got)
			}
		})
	}

	// The same parameters are the same name, at this launch and the next.
	same := fsrs.DefaultParam()
	same.EnableFuzz = false
	if got := weighed(same); got != was {
		t.Errorf("the same parameters are named %q and then %q", was, got)
	}
}
