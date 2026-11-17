// Package machine reads what the machine gives the process.
package machine

import "runtime"

// GetCores returns how many processors the process may use at once.
func GetCores() int {
	// gomutants:disable-next-line INTEGER_DECREMENT reason="Any non-positive argument queries GOMAXPROCS without changing it."
	return runtime.GOMAXPROCS(0)
}
