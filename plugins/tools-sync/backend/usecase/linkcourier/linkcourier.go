// Package linkcourier carries the blobs of a link between the core of this device and the relay of the sync server.
package linkcourier

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Deps are what the courier needs.
type Deps struct {
	Core   ports.LinkCore
	Relay  ports.Relay
	Health ports.Health
	Clock  ports.Clock
}

// Courier runs for as long as the plugin does.
type Courier struct{ deps Deps }

// NewCourier returns the courier over the dependencies.
func NewCourier(deps Deps) *Courier { return &Courier{deps: deps} }

// Execute carries work until ctx ends.
func (u *Courier) Execute(ctx context.Context) error { return errors.New("not implemented") }
