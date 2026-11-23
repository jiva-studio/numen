// Package watchchanges keeps the stream of change hints open and wakes the sync of each vault a hint names.
package watchchanges

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Deps are the ports WatchChanges needs.
type Deps struct {
	Hints ports.Hints
	Host  ports.Host
	Links ports.Links
	Waker ports.Waker
	Clock ports.Clock
}

// WatchChanges reads the stream of hints until its context ends.
type WatchChanges struct {
	deps Deps
}

// NewWatchChanges returns the use case over deps.
func NewWatchChanges(deps Deps) *WatchChanges { return &WatchChanges{deps: deps} }

// Execute opens the stream, wakes the vaults it names and opens it again after each end, until ctx ends.
func (u *WatchChanges) Execute(context.Context) error { return errors.New("not implemented") }
