// Package listing reads the models and the reasoning levels an agent program reports, and keeps them.
package listing

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/agents"
)

// listLimit is how long a program has to open the session its models are read from.
const listLimit = 15 * time.Second

// failureBackoff is how long a program that failed to list is left alone.
const failureBackoff = 30 * time.Second

// Lister reads the listing of a program from a session and keeps it.
type Lister struct {
	launcher acp.Launcher
	listings *cache.Cache[acp.Listing]
}

// New returns a Lister that starts programs with launcher. When isAsync is set, a program whose listing
// is not kept yet is listed with none and read in the background.
func New(launcher acp.Launcher, source cache.Clock, isAsync bool) (*Lister, error) {
	if launcher == nil || source == nil {
		return nil, errors.New("listing: a launcher and a clock are required")
	}
	listings := cache.New[acp.Listing](
		"models",
		cache.WithTimeout(listLimit),
		cache.WithFailureBackoff(failureBackoff),
		cache.WithAsync(isAsync),
		cache.WithClock(source),
	)
	return &Lister{launcher: launcher, listings: listings}, nil
}

// GetListing returns the models the program of the agent reports, from the cache when they are kept. A
// failed read leaves the last listing of the cache, or none; a session that fails to close fails the read.
func (l *Lister) GetListing(ctx context.Context, agent agents.Agent) acp.Listing {
	listing, _, err := l.listings.Get(ctx, agent.GetKey(), func(ctx context.Context) (acp.Listing, error) {
		return l.readListing(ctx, agent)
	})
	if err != nil {
		return listing
	}
	return listing
}

func (l *Lister) readListing(ctx context.Context, agent agents.Agent) (listing acp.Listing, err error) {
	asking, stop := context.WithTimeout(ctx, listLimit)
	defer stop()
	session, err := acp.Open(asking, l.launcher, acp.Start{Command: agent.Command, Servers: nil})
	if err != nil {
		return acp.Listing{}, err
	}
	defer func() { err = errors.Join(err, session.Close(ctx)) }()
	return session.GetListing(), nil
}
