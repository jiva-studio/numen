package logs

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// History reads recent log entries stored on disk for a day.
type History interface {
	ReadRecent(day time.Time, limit int) ([]logs.Entry, error)
}

// Watch streams recent and incoming structured log entries until the caller's context ends.
type Watch struct {
	entries *Entries
	history History
	clock   machineport.Clock
}

// NewWatch returns the use case over the in-memory buffer, the persistent history and the clock.
func NewWatch(entries *Entries, history History, clock machineport.Clock) *Watch {
	return &Watch{
		entries: entries,
		history: history,
		clock:   clock,
	}
}

// Execute loads recent entries from today's log file and streams buffered and live entries.
func (w *Watch) Execute(ctx context.Context) <-chan []logs.Entry {
	var recent []logs.Entry
	if w.history != nil && w.clock != nil {
		if loaded, err := w.history.ReadRecent(w.clock.Now(), MaxBufferedEntries); err == nil {
			recent = loaded
		}
	}
	return w.entries.Watch(ctx, recent)
}
