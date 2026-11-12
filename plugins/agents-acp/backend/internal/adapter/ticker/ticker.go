// Package ticker runs a function at a fixed period on the machine clock.
package ticker

import (
	"sync"
	"time"
)

// Start calls run every period until the returned function is called. That function waits for a call in
// progress to return, and returns the error of the last call that failed.
func Start(period time.Duration, run func() error) func() error {
	ticks := time.NewTicker(period)
	done := make(chan struct{})
	stopped := make(chan struct{})
	var lastErr error
	go func() {
		defer close(stopped)
		for {
			select {
			case <-done:
				return
			case <-ticks.C:
				if err := run(); err != nil {
					lastErr = err
				}
			}
		}
	}()
	var once sync.Once
	return func() error {
		once.Do(func() {
			ticks.Stop()
			close(done)
			<-stopped
		})
		return lastErr
	}
}
