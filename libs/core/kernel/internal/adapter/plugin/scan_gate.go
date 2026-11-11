package plugin

import "sync"

// scanGate keeps a scan of the plugin folders and the start of a plugin apart, so that a plugin is not
// started while the set of plugins changes. A scan holds it for writing and a start holds it for reading.
type scanGate struct {
	mu sync.RWMutex
}

func (g *scanGate) holdScan()     { g.mu.Lock() }
func (g *scanGate) releaseScan()  { g.mu.Unlock() }
func (g *scanGate) holdStart()    { g.mu.RLock() }
func (g *scanGate) releaseStart() { g.mu.RUnlock() }
