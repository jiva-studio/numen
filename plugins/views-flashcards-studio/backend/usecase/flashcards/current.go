package flashcards

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
)

// Scenarios is everything the plugin does for a session of review, built for one day start.
type Scenarios struct {
	Reviews   ReviewScenarios
	Schedules ScheduleScenarios
	Counts    CountScenarios
	Day       review.Day
}

// Current holds the review scenarios in force. The host can change the day a review begins at any
// time while calls run, and the scenarios are built again for it; calls already running finish under
// the scenarios they began under.
type Current struct {
	config Config
	active atomic.Pointer[Scenarios]
}

// NewCurrent builds the review scenarios from the config and holds them. A count of cores that is not
// positive is the cores the process may use.
func NewCurrent(cfg Config) *Current {
	if cfg.Cores <= 0 {
		// gomutants:disable-next-line INTEGER_DECREMENT reason="Any non-positive argument queries GOMAXPROCS without changing it."
		cfg.Cores = runtime.GOMAXPROCS(0)
	}
	c := &Current{config: cfg, active: atomic.Pointer[Scenarios]{}}
	c.active.Store(createScenarios(cfg))
	return c
}

func (c *Current) Get() *Scenarios { return c.active.Load() }

// SetDayStart builds the review scenarios again with the time past midnight at which a day of review begins.
func (c *Current) SetDayStart(starts time.Duration) {
	cfg := c.config
	cfg.Day = review.Day{Starts: starts, In: nil}
	c.active.Store(createScenarios(cfg))
}

func createScenarios(cfg Config) *Scenarios {
	reviews, schedules, counts := NewScenarios(cfg)
	return &Scenarios{Reviews: reviews, Schedules: schedules, Counts: counts, Day: cfg.Day}
}
