package plugin

import (
	"slices"
	"time"
)

// Leases are the leases on one running plugin process: who holds them, who started the process, when a use
// lease was last taken or returned, and whether one was taken since the process started. A question lease
// holds the process as a use lease does, but neither marks it used nor moves LastUseTime. The zero value
// holds nothing. Leases is not safe for concurrent use.
type Leases struct {
	holders     []Holder
	startedBy   Holder
	lastUseTime time.Time
	isUsed      bool
}

// LeaseState is what Leases hold at one moment.
type LeaseState struct {
	Holders     []Holder
	StartedBy   Holder
	LastUseTime time.Time
	IsUsed      bool
}

// Begin records the starter of a process that is starting.
func (l *Leases) Begin(starter Holder) {
	l.startedBy = starter
}

// Start marks the process started at now and not used since.
func (l *Leases) Start(now time.Time) {
	l.lastUseTime = now
	l.isUsed = false
}

// StartHeld marks the process started at now and held by the holder alone.
func (l *Leases) StartHeld(holder Holder, isUse bool, now time.Time) {
	l.holders = []Holder{holder}
	l.lastUseTime = now
	l.isUsed = isUse
}

// Add records a lease of the holder taken at now.
func (l *Leases) Add(holder Holder, isUse bool, now time.Time) {
	l.holders = append(l.holders, holder)
	if isUse {
		l.lastUseTime = now
		l.isUsed = true
	}
}

// Remove records a lease of the holder returned at now, and reports whether no lease is left. Where the
// holder holds none, the oldest lease is removed.
func (l *Leases) Remove(holder Holder, isUse bool, now time.Time) bool {
	if len(l.holders) > 0 {
		i := max(slices.Index(l.holders, holder), 0)
		l.holders = slices.Delete(l.holders, i, i+1)
	}
	if isUse {
		l.lastUseTime = now
	}
	return len(l.holders) == 0
}

// Clear drops every lease, as for a process that ended.
func (l *Leases) Clear() {
	l.holders = nil
}

// Count returns the number of leases held.
func (l *Leases) Count() int {
	return len(l.holders)
}

// IsDueToStop reports whether a process that holds no lease stops at now: one used since it started once
// it has been idle for threshold, and one never used at once.
func (l *Leases) IsDueToStop(threshold time.Duration, now time.Time) bool {
	return len(l.holders) == 0 && (!l.isUsed || now.Sub(l.lastUseTime) >= threshold)
}

// GetState returns what the leases hold now.
func (l *Leases) GetState() LeaseState {
	return LeaseState{
		Holders: slices.Clone(l.holders), StartedBy: l.startedBy, LastUseTime: l.lastUseTime, IsUsed: l.isUsed,
	}
}
