// Package conflict settles a file both computers changed, with the strategy its kind has.
package conflict

import (
	"context"
	"log/slog"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

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
func New(deps Deps) *Resolver {
	if deps.Log == nil {
		deps.Log = slog.New(slog.DiscardHandler)
	}
	return &Resolver{deps: deps}
}

// Pick returns the strategy of the path: a copy and the head for a note, the head alone for any other file.
func (r *Resolver) Pick(path string) Strategy {
	if note.Kind(path) == note.KindMarkdown {
		return markdown{host: r.deps.Host}
	}
	return overwrite{log: r.deps.Log}
}

// Resolve settles the conflict with the strategy of its path.
func (r *Resolver) Resolve(ctx context.Context, in ports.ConflictInput) (ports.ConflictOutcome, error) {
	return r.Pick(in.Path).Resolve(ctx, in)
}
