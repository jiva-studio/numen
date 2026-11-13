package connection

import (
	"sync"
	"time"
)

const (
	// DefaultCooldown is how long a connection is set aside when the service gave no wait.
	DefaultCooldown = 60 * time.Second
	// MaxCooldown is the longest a connection is set aside for a wait the service asked for.
	MaxCooldown = 10 * time.Minute
)

type version struct {
	id       string
	revision int
}

type cooldownEntry struct {
	until     time.Time
	isForever bool
	reason    FailureReason
}

type notice struct {
	version version
	reason  FailureReason
}

// Cooldowns are the connections set aside, by id and revision, and the notices already given for them. An
// edit changes the revision, so the entry of the old revision no longer matches. A rejected key or a payment
// required sets a connection aside until it is edited; a built-in connection cannot be edited, so its
// cooldown always ends in time. A new cooldown of a connection allows a new notice. Cooldowns are kept in
// memory, so a restart allows the notices again. Cooldowns are safe for concurrent use.
type Cooldowns struct {
	mu      sync.Mutex
	entries map[version]cooldownEntry
	claimed map[notice]bool
}

// NewCooldowns returns cooldowns that set nothing aside.
func NewCooldowns() *Cooldowns {
	return &Cooldowns{entries: map[version]cooldownEntry{}, claimed: map[notice]bool{}}
}

// Get returns the reason the revision of the connection is set aside at now, and false where it is not.
func (c *Cooldowns) Get(id string, revision int, now time.Time) (FailureReason, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := version{id: id, revision: revision}
	entry, found := c.entries[key]
	if !found {
		return entry.reason, false
	}
	if !entry.isForever && !now.Before(entry.until) {
		delete(c.entries, key)
		return entry.reason, false
	}
	return entry.reason, true
}

// Start sets the revision of the connection aside from now for the reason: for retryAfter, at most
// MaxCooldown, where the service asked for a wait, and for DefaultCooldown where it did not.
func (c *Cooldowns) Start(id string, revision int, reason FailureReason, retryAfter time.Duration, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := version{id: id, revision: revision}
	for other := range c.entries {
		if other.id == id && other.revision != revision {
			delete(c.entries, other)
		}
	}
	entry := cooldownEntry{reason: reason}
	switch {
	case lastsUntilEdited(reason) && !IsDetected(id):
		entry.isForever = true
	case retryAfter > 0:
		entry.until = now.Add(min(retryAfter, MaxCooldown))
	default:
		entry.until = now.Add(DefaultCooldown)
	}
	for given := range c.claimed {
		if given.version == key {
			delete(c.claimed, given)
		}
	}
	c.entries[key] = entry
}

// Claim reports whether the notice of the revision of the connection set aside for the reason is still to
// be given, and marks it given.
func (c *Cooldowns) Claim(id string, revision int, reason FailureReason) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	one := notice{version: version{id: id, revision: revision}, reason: reason}
	if c.claimed[one] {
		return false
	}
	c.claimed[one] = true
	return true
}

func lastsUntilEdited(reason FailureReason) bool {
	return reason == FailureReasonRejectedKey || reason == FailureReasonPaymentRequired
}
