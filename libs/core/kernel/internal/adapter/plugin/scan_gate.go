package plugin

import "sync"

// ScanGate keeps a scan of the plugin folders and the start of a plugin apart, so that a plugin is not
// started while the set of plugins changes. A scan holds it for writing and a start holds it for reading.
type ScanGate struct {
	mu sync.RWMutex
}

// NewScanGate creates a gate that no scan and no start holds.
func NewScanGate() *ScanGate { return &ScanGate{} }

func (g *ScanGate) holdScan()     { g.mu.Lock() }
func (g *ScanGate) releaseScan()  { g.mu.Unlock() }
func (g *ScanGate) holdStart()    { g.mu.RLock() }
func (g *ScanGate) releaseStart() { g.mu.RUnlock() }
