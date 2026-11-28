package loops

import "sync/atomic"

// Flag is a switch that one goroutine sets and others read.
type Flag struct{ on atomic.Bool }

func (f *Flag) Set(on bool) { f.on.Store(on) }

func (f *Flag) IsOn() bool { return f.on.Load() }
