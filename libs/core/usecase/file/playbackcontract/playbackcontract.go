// Package playbackcontract re-exports the playback port contract.
package playbackcontract

import "github.com/jiva-studio/numen/libs/core/port/playbackport"

type (
	Copies = playbackport.Copies
	Copy   = playbackport.Copy
)

var (
	ErrNoProxy = playbackport.ErrNoProxy
)
