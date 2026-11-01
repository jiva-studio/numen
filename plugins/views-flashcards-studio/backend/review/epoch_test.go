package review_test

import (
	"time"
)

// testEpoch is the moment that the tests use as the current time.
func getTestEpoch() time.Time {
	return time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)
}
