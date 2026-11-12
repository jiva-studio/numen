package modelcall

import (
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

type connectionVersion struct {
	id       string
	revision int
}

type cooldownEntry struct {
	until     time.Time
	isForever bool
	reason    connection.FailureReason
}

// cooldown is the connections set aside, by id and revision, and the notices already given for them. An edit
// changes the revision, so the entry of the old revision no longer matches. A new cooldown of a connection
// allows a new notice. It is kept in memory, so a restart allows the notices again.
type cooldown struct {
	mu      sync.Mutex
	entries map[connectionVersion]cooldownEntry
	claimed map[notice]bool
}

func newCooldown() *cooldown {
	return &cooldown{entries: map[connectionVersion]cooldownEntry{}, claimed: map[notice]bool{}}
}

func (c *cooldown) get(version connectionVersion, now time.Time) (cooldownEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, found := c.entries[version]
	if !found {
		return cooldownEntry{}, false
	}
	if !entry.isForever && !now.Before(entry.until) {
		delete(c.entries, version)
		return cooldownEntry{}, false
	}
	return entry, true
}

func (c *cooldown) start(version connectionVersion, reason connection.FailureReason, retryAfter time.Duration, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for other := range c.entries {
		if other.id == version.id && other.revision != version.revision {
			delete(c.entries, other)
		}
	}
	entry := cooldownEntry{reason: reason}
	switch {
	case lastsUntilEdited(reason):
		entry.isForever = true
	case retryAfter > 0:
		entry.until = now.Add(min(retryAfter, MaxCooldown))
	default:
		entry.until = now.Add(DefaultCooldown)
	}
	for given := range c.claimed {
		if given.version == version {
			delete(c.claimed, given)
		}
	}
	c.entries[version] = entry
}

func lastsUntilEdited(reason connection.FailureReason) bool {
	return reason == connection.FailureReasonRejectedKey || reason == connection.FailureReasonPaymentRequired
}

type notice struct {
	version connectionVersion
	reason  connection.FailureReason
}

func (c *cooldown) claim(one notice) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.claimed[one] {
		return false
	}
	c.claimed[one] = true
	return true
}
