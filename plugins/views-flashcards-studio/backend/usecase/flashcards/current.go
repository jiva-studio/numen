package flashcards

import (
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
)

// UseCases is everything the plugin does for a session of review, built for one day start.
type UseCases[P any] struct {
	Reviews   ReviewUseCases
	Schedules ScheduleUseCases[P]
	Counts    CountUseCases
	Day       review.Day
}

// Current holds the review use cases in force. The host can change the day a review begins at any
// time while calls run, and the use cases are built again for it; calls already running finish under
// the use cases they began under.
type Current[P any] struct {
	config     Config
	newPresets NewPresets[P]
	active     atomic.Pointer[UseCases[P]]
}

// NewCurrent builds the review use cases from the config and the presets newPresets makes, and
// holds them.
func NewCurrent[P any](cfg Config, newPresets NewPresets[P]) *Current[P] {
	c := &Current[P]{config: cfg, newPresets: newPresets, active: atomic.Pointer[UseCases[P]]{}}
	c.active.Store(c.createUseCases(cfg))
	return c
}

// Get returns the review use cases in force.
func (c *Current[P]) Get() *UseCases[P] { return c.active.Load() }

// SetDayStart builds the review use cases again with the time past midnight at which a day of review begins.
func (c *Current[P]) SetDayStart(starts time.Duration) {
	cfg := c.config
	cfg.Day = review.Day{Starts: starts, In: nil}
	c.active.Store(c.createUseCases(cfg))
}

func (c *Current[P]) createUseCases(cfg Config) *UseCases[P] {
	reviews, schedules, counts := NewUseCases(cfg, c.newPresets)
	return &UseCases[P]{Reviews: reviews, Schedules: schedules, Counts: counts, Day: cfg.Day}
}
