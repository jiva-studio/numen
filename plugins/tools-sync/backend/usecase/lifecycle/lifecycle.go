// Package lifecycle keeps the sync of a phone to the time the application is on the screen.
package lifecycle

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Grace is how long Suspend waits for the last pass.
const Grace = 20 * time.Second

// Pass is one pass over one vault.
type Pass interface {
	Execute(ctx context.Context, id domain.VaultID) error
}

// Loop is the interval loop over the connected vaults. It runs until its context ends.
type Loop interface {
	Execute(ctx context.Context) error
}

// Stream is the stream of hints of the server. Watch returns when its context ends, and calls changed with
// the vault each hint names.
type Stream interface {
	Watch(ctx context.Context, changed func(domain.VaultID)) error
}

// Deps are what the use case needs.
type Deps struct {
	Pass   Pass
	Loop   Loop
	Stream Stream
	Host   ports.Host
	State  ports.Links
	Clock  ports.Clock
}

// Lifecycle starts and stops the loops of the sync as the application comes to the screen and leaves it.
type Lifecycle struct{ deps Deps }

// NewLifecycle returns the use case over the dependencies.
func NewLifecycle(deps Deps) *Lifecycle { return &Lifecycle{deps: deps} }

// Resume runs a pass over every connected vault at once and starts the loop and the stream. A second call
// while they run does nothing.
func (u *Lifecycle) Resume(context.Context) error { return errors.New("not implemented") }

// Suspend stops the loop and the stream, and runs one pass over every connected vault within Grace. A second
// call while they are stopped does nothing. It returns when nothing of the sync runs.
func (u *Lifecycle) Suspend(context.Context) error { return errors.New("not implemented") }
