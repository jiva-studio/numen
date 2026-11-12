// Package ticker runs a function at a fixed period on the machine clock.
package ticker

import (
	"sync"
	"time"
)

// Start calls run every period until the returned function is called, which waits for a call in
// progress to return.
func Start(period time.Duration, run func()) func() {
	ticks := time.NewTicker(period)
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			select {
			case <-done:
				return
			case <-ticks.C:
				run()
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			ticks.Stop()
			close(done)
			<-stopped
		})
	}
}
