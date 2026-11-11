package report

import (
	"sync"
	"time"
)

// SpeedWindow is how far back the speed of embedding is measured.
const SpeedWindow = 10 * time.Second

// minSpan is the shortest time two observations are apart for the speed between them to mean
// anything.
const minSpan = time.Second

// Counts are the numbers of the index the meter watches.
type Counts struct {
	Vectors int64
	Chunks  int64
	Sources int64
}

// Reading is what the meter derives from the observations so far.
type Reading struct {
	Speed      float64
	HasSpeed   bool
	ChangedAt  time.Time
	HasChanged bool
}

type sample struct {
	takenAt time.Time
	vectors int64
}

// Meter measures the speed of embedding and notes the time of the last change of the counts. Its
// observations come from the calls of the view, so it knows only the changes that happened while the
// view asked.
type Meter struct {
	mu         sync.Mutex
	samples    []sample
	last       Counts
	hasLast    bool
	changedAt  time.Time
	hasChanged bool
}

// NewMeter returns a meter that has observed nothing.
func NewMeter() *Meter {
	return &Meter{
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="m.last is unread while hasLast is false."
		mu: sync.Mutex{}, samples: nil, last: Counts{Vectors: 0, Chunks: 0, Sources: 0}, hasLast: false,
		changedAt: time.Time{}, hasChanged: false,
	}
}

// Observe records the counts seen at the time and returns what the meter derives. The speed is the
// rise of the vector count between the oldest observation inside the window and this one; a fall of
// the count, from a deleted source, is no speed.
func (m *Meter) Observe(now time.Time, counts Counts) Reading {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hasLast && counts != m.last {
		m.changedAt, m.hasChanged = now, true
	}
	m.last, m.hasLast = counts, true

	derived := Reading{Speed: 0, HasSpeed: false, ChangedAt: m.changedAt, HasChanged: m.hasChanged}
	if counts.Chunks <= counts.Vectors {
		m.samples = nil
		return derived
	}
	m.samples = append(m.keepRecent(now), sample{takenAt: now, vectors: counts.Vectors})
	oldest := m.samples[0]
	span := now.Sub(oldest.takenAt)
	if span < minSpan {
		return derived
	}
	derived.Speed = max(0, float64(counts.Vectors-oldest.vectors)/span.Seconds())
	derived.HasSpeed = true
	return derived
}

func (m *Meter) keepRecent(now time.Time) []sample {
	recent := m.samples[:0]
	for _, one := range m.samples {
		if now.Sub(one.takenAt) <= SpeedWindow {
			recent = append(recent, one)
		}
	}
	return recent
}
