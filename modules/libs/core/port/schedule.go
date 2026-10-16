package port

import (
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// ScheduleAlgorithm produces review schedulers for given retention targets.
type ScheduleAlgorithm interface {
	GetName() string
	CreateScheduler(retention float64) review.Scheduler
}
