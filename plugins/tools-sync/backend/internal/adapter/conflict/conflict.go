// Package conflict settles a file both computers changed, with the strategy its kind has.
package conflict

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

var errNotImplemented = errors.New("not implemented")

// Strategy settles a conflict of one kind of file.
type Strategy interface {
	Resolve(ctx context.Context, in ports.ConflictInput) (ports.ConflictOutcome, error)
}

// Deps are what the resolver needs. A nil Log writes nowhere.
type Deps struct {
	Host ports.Host
	Log  *slog.Logger
}

// Resolver picks the strategy of a path and resolves with it.
type Resolver struct{ deps Deps }

// New returns the resolver over the dependencies.
func New(deps Deps) *Resolver { return &Resolver{deps: deps} }

// Pick returns the strategy of the path.
func (r *Resolver) Pick(string) Strategy { return stub{} }

// Resolve settles the conflict with the strategy of its path.
func (r *Resolver) Resolve(ctx context.Context, in ports.ConflictInput) (ports.ConflictOutcome, error) {
	return r.Pick(in.Path).Resolve(ctx, in)
}

type stub struct{}

func (stub) Resolve(context.Context, ports.ConflictInput) (ports.ConflictOutcome, error) {
	return ports.ConflictOutcome{Applied: nil, Copy: ""}, errNotImplemented
}
