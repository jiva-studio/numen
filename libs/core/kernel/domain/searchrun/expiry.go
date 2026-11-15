package searchrun

import (
	"slices"
	"time"
)

// IsExpired reports whether a record was not read for the TTL at now.
func IsExpired(r Record, now time.Time, policy Policy) bool {
	return now.Sub(r.AccessedAt) >= policy.TTL
}

// SelectExpired returns the searches to drop at now: every record not read for the TTL, then, while more
// than MaxKept would remain with incoming new ones, the stopped records read least recently. A running
// record is dropped by the TTL only.
func SelectExpired(records []Record, now time.Time, policy Policy, incoming int) []ID {
	var expired []ID
	var stopped []Record
	kept := 0
	for _, r := range records {
		switch {
		case IsExpired(r, now, policy):
			expired = append(expired, r.ID)
		case !r.IsRunning:
			stopped = append(stopped, r)
			kept++
		default:
			kept++
		}
	}
	excess := min(kept+incoming-policy.MaxKept, len(stopped))
	if excess <= 0 {
		return expired
	}
	slices.SortStableFunc(stopped, func(a, b Record) int { return a.AccessedAt.Compare(b.AccessedAt) })
	for _, r := range stopped[:excess] {
		expired = append(expired, r.ID)
	}
	return expired
}
