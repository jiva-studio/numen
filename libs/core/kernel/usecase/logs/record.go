package logs

import (
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Store persists a structured log entry.
type Store interface {
	WriteEntry(entry logs.Entry) error
}

// Record validates, filters by the active log level, buffers and persists a structured log entry.
type Record struct {
	entries  *Entries
	store    Store
	getLevel func() logs.Level
	clock    machineport.Clock
}

// NewRecord returns the use case over the in-memory buffer, the file store, the active log level
// reader and the clock.
func NewRecord(entries *Entries, store Store, level func() logs.Level, clock machineport.Clock) *Record {
	return &Record{
		entries:  entries,
		store:    store,
		getLevel: level,
		clock:    clock,
	}
}

// Execute normalizes entry, drops it when its severity is below the configured level, appends it to
// the in-memory stream and writes it to the store.
func (r *Record) Execute(entry logs.Entry) error {
	activeLevel := logs.DefaultLevel
	if r.getLevel != nil {
		activeLevel = r.getLevel()
	}
	if !activeLevel.CanRecord(entry.Level) {
		if strings.TrimSpace(entry.Message) == "" {
			return logs.ErrEmptyMessage
		}
		return nil
	}
	if entry.Time.IsZero() && r.clock != nil {
		entry.Time = r.clock.Now()
	}
	normalized, err := logs.Normalize(entry)
	if err != nil {
		return err
	}
	if r.entries != nil {
		r.entries.Append(normalized)
	}
	if r.store != nil {
		return r.store.WriteEntry(normalized)
	}
	return nil
}
