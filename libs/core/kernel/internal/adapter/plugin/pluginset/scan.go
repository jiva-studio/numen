package pluginset

import "sync"

// ScanLock keeps a scan of the plugin folders and the start of a plugin apart, so that a plugin is not
// started while the set of plugins changes. A scan holds it for writing and a start holds it for reading.
type ScanLock struct {
	mu sync.RWMutex
}

// BeginScan holds the lock for a scan of the plugin folders, until EndScan.
func (l *ScanLock) BeginScan() { l.mu.Lock() }

// EndScan ends the hold BeginScan took.
func (l *ScanLock) EndScan() { l.mu.Unlock() }

// BeginActivation holds the lock for the start of a plugin, until EndActivation.
func (l *ScanLock) BeginActivation() { l.mu.RLock() }

// EndActivation ends the hold BeginActivation took.
func (l *ScanLock) EndActivation() { l.mu.RUnlock() }
