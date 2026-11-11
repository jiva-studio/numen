// Package scenarios holds the review scenarios in force. The settings can change the day a review
// begins, and the scenarios are built again for it.
package scenarios

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/flashcards"
)

// Review is everything the plugin does for a session of review, built for one day start.
type Review struct {
	Reviews   flashcards.ReviewScenarios
	Schedules flashcards.ScheduleScenarios
	Counts    flashcards.CountScenarios
	Day       review.Day
}

// Current holds the review scenarios in force. Calls already running finish under the scenarios they
// began under.
type Current struct {
	config flashcards.Config
	active atomic.Pointer[Review]
}

// NewCurrent builds the review scenarios from the config and holds them. A count of cores that is not
// positive is the cores the process may use.
func NewCurrent(cfg flashcards.Config) *Current {
	if cfg.Cores <= 0 {
		// gomutants:disable-next-line INTEGER_DECREMENT reason="Any non-positive argument queries GOMAXPROCS without changing it."
		cfg.Cores = runtime.GOMAXPROCS(0)
	}
	c := &Current{config: cfg, active: atomic.Pointer[Review]{}}
	c.active.Store(build(cfg))
	return c
}

func (c *Current) Get() *Review { return c.active.Load() }

// SetDayStart builds the review scenarios again with the time past midnight at which a day of review begins.
func (c *Current) SetDayStart(starts time.Duration) {
	cfg := c.config
	cfg.Day = review.Day{Starts: starts, In: nil}
	c.active.Store(build(cfg))
}

func build(cfg flashcards.Config) *Review {
	reviews, schedules, counts := flashcards.NewScenarios(cfg)
	return &Review{Reviews: reviews, Schedules: schedules, Counts: counts, Day: cfg.Day}
}
