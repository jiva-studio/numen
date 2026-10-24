package signal

import "runtime"

// GetWorkerCount is how many goroutines share jobs units of work: the count
// asked for, or one per CPU when none is, and never more than there are jobs.
func GetWorkerCount(requested, jobs int) int {
	if requested <= 0 {
		requested = runtime.GOMAXPROCS(0)
	}
	return max(1, min(requested, jobs))
}
