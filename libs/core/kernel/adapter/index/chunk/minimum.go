package chunk

import "sync/atomic"

// Minimum is the length below which a small chunk gets no vector, shared by what selects the
// chunks to embed and what counts them, so both answer for the same set.
//
// Zero gives every small chunk a vector. The length is in bytes of the chunk's text, as `length`
// is stored.
type Minimum struct{ length atomic.Int64 }

// Get returns the shortest length that gets a vector.
func (m *Minimum) Get() int64 { return m.length.Load() }

// Swap sets the shortest length that gets a vector and returns the one it replaces.
func (m *Minimum) Swap(length int64) int64 { return m.length.Swap(length) }
