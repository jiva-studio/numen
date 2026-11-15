package searchrun

import "time"

// Policy bounds the searches. TTL is how long a search is kept after its last read, MaxKept how many are
// kept, MaxRunning how many run at once, MaxWait the longest wait of one read, DefaultPage and MaxPage the
// results of one page.
type Policy struct {
	TTL         time.Duration
	MaxKept     int
	MaxRunning  int
	MaxWait     time.Duration
	DefaultPage int
	MaxPage     int
}

// DefaultPolicy returns the policy the core runs with.
func DefaultPolicy() Policy {
	return Policy{
		TTL: 10 * time.Minute, MaxKept: 20, MaxRunning: 4, MaxWait: 30 * time.Second, DefaultPage: 20, MaxPage: 100,
	}
}

// GetPageSize returns the results of one page for a requested size.
func (p Policy) GetPageSize(size int) int {
	if size <= 0 {
		return p.DefaultPage
	}
	return min(size, p.MaxPage)
}
