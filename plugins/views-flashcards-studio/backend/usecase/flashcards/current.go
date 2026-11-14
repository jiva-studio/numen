package flashcards

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
)

// Review is everything the plugin does for a session of review, built for one day start.
type Review[P DeckPresets] struct {
	Reviews   ReviewScenarios
	Schedules ScheduleScenarios[P]
	Counts    CountScenarios
	Day       review.Day
}

// Current holds the review scenarios in force. The settings can change the day a review begins,
// and the scenarios are built again for it. Calls already running finish under the scenarios they
// began under.
type Current[P DeckPresets] struct {
	config     Config
	newPresets func(index IndexUpdate, day review.Day) P
	active     atomic.Pointer[Review[P]]
}

// NewCurrent builds the review scenarios from the config and the presets newPresets makes, and
// holds them. A count of cores that is not positive is the cores the process may use.
func NewCurrent[P DeckPresets](cfg Config, newPresets func(index IndexUpdate, day review.Day) P) *Current[P] {
	if cfg.Cores <= 0 {
		// gomutants:disable-next-line INTEGER_DECREMENT reason="Any non-positive argument queries GOMAXPROCS without changing it."
		cfg.Cores = runtime.GOMAXPROCS(0)
	}
	c := &Current[P]{config: cfg, newPresets: newPresets, active: atomic.Pointer[Review[P]]{}}
	c.active.Store(c.build(cfg))
	return c
}

// Get returns the review scenarios in force.
func (c *Current[P]) Get() *Review[P] { return c.active.Load() }

// SetDayStart builds the review scenarios again with the time past midnight at which a day of review begins.
func (c *Current[P]) SetDayStart(starts time.Duration) {
	cfg := c.config
	cfg.Day = review.Day{Starts: starts, In: nil}
	c.active.Store(c.build(cfg))
}

func (c *Current[P]) build(cfg Config) *Review[P] {
	reviews, schedules, counts := NewScenarios(cfg, c.newPresets)
	return &Review[P]{Reviews: reviews, Schedules: schedules, Counts: counts, Day: cfg.Day}
}
