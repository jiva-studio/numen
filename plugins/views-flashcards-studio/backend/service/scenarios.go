package service

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/flashcards"
)

// reviewScenarios is everything the plugin does for a session of review, built for one day start.
type reviewScenarios struct {
	Reviews   flashcards.ReviewScenarios
	Schedules flashcards.ScheduleScenarios
	Counts    flashcards.CountScenarios
	Day       review.Day
}

// currentScenarios holds the review scenarios in force. The settings can change the day a review
// begins, and the scenarios are built again for it. Calls already running finish under the
// scenarios they began under.
type currentScenarios struct {
	config flashcards.Config
	active atomic.Pointer[reviewScenarios]
}

// createCurrentScenarios builds the review scenarios from the config and holds them. A count of cores
// that is not positive is the cores the process may use.
func createCurrentScenarios(cfg flashcards.Config) *currentScenarios {
	if cfg.Cores <= 0 {
		// gomutants:disable-next-line INTEGER_DECREMENT reason="Any non-positive argument queries GOMAXPROCS without changing it."
		cfg.Cores = runtime.GOMAXPROCS(0)
	}
	c := &currentScenarios{config: cfg, active: atomic.Pointer[reviewScenarios]{}}
	c.active.Store(createScenarios(cfg))
	return c
}

func (c *currentScenarios) Get() *reviewScenarios { return c.active.Load() }

// SetDayStart builds the review scenarios again with the time past midnight at which a day of review begins.
func (c *currentScenarios) SetDayStart(starts time.Duration) {
	cfg := c.config
	cfg.Day = review.Day{Starts: starts, In: nil}
	c.active.Store(createScenarios(cfg))
}

func createScenarios(cfg flashcards.Config) *reviewScenarios {
	reviews, schedules, counts := flashcards.NewScenarios(cfg)
	return &reviewScenarios{Reviews: reviews, Schedules: schedules, Counts: counts, Day: cfg.Day}
}
