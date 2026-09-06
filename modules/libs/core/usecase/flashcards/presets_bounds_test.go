package flashcards_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// Settings a preset may not hold are refused, the reason names the key, and the
// note is left as the person wrote it.
//
// A client that named none of goal, learned or counts sends a value that is not
// one of them, and a day the goal names has to be there for the goal to read.
func TestSettingsAPresetMayNotHoldAreRefused(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		what   string
		of     func(review.Preset) review.Preset
		reason string
	}{{
		what:   "a goal the application does not know",
		of:     func(p review.Preset) review.Preset { p.Goal = ""; return p },
		reason: "goal",
	}, {
		what:   "a rule the application does not know",
		of:     func(p review.Preset) review.Preset { p.Rule = ""; return p },
		reason: "learned",
	}, {
		what:   "a goal of a date naming no day",
		of:     func(p review.Preset) review.Preset { p.Goal, p.By = review.GoalDate, time.Time{}; return p },
		reason: "day",
	}, {
		what: "a share of a day outside its bounds",
		of: func(p review.Preset) review.Preset {
			p.Load = map[time.Weekday]int{time.Saturday: 140}
			return p
		},
		reason: "the load of sat",
	}} {
		t.Run(one.what, func(t *testing.T) {
			s := opened(t, settled)
			was := read(t, s.vault, "Sanskrit.md")

			_, err := s.presets.Save(t.Context(), s.vault, "Sanskrit.md", one.of(minutes()), domain.Fingerprint{})
			if !errors.Is(err, flashcards.ErrOutOfBounds) {
				t.Fatalf("saving %s said %v", one.what, err)
			}
			if !strings.Contains(err.Error(), one.reason) {
				t.Errorf("the refusal of %s says %q, and names nothing of %q", one.what, err, one.reason)
			}
			if held := read(t, s.vault, "Sanskrit.md"); held != was {
				t.Errorf("the note was written:\n%s", held)
			}
		})
	}
}
