// Package chaincourier keeps the chain of the core and the chain of the sync server the same.
package chaincourier

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Deps are what the courier needs.
type Deps struct {
	Identity ports.Identity
	Core     ports.ChainCore
	Chain    ports.Chain
}

// Publish gives the server the chain the core wants it to hold and tells the core the answer.
type Publish struct{ deps Deps }

// NewPublish returns the use case over the dependencies.
func NewPublish(deps Deps) *Publish { return &Publish{deps: deps} }

// Execute publishes the pending chain of the core. It returns whether a chain was pending.
func (u *Publish) Execute(ctx context.Context) (bool, error) {
	return false, errors.New("not implemented")
}

// Adopt gives the core the chain of the server when the server holds a longer one than the core.
type Adopt struct{ deps Deps }

// NewAdopt returns the use case over the dependencies.
func NewAdopt(deps Deps) *Adopt { return &Adopt{deps: deps} }

// Execute returns whether the core took a chain.
func (u *Adopt) Execute(ctx context.Context) (bool, error) {
	return false, errors.New("not implemented")
}
