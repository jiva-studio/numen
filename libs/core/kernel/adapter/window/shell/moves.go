package shell

import (
	"context"
)

// ReportChangesFrom calls ReportChange for every signal on the channel, until
// it closes or ctx is done.
func (a *API) ReportChangesFrom(ctx context.Context, moved <-chan struct{}) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case _, open := <-moved:
				if !open {
					return
				}
				a.Changes.ReportChange()
			}
		}
	}()
}

// WatchIndexChanges signals that something moved, until ctx is done. Plugins of the
// installation receive it through the host.
func (a *API) WatchIndexChanges(ctx context.Context) (<-chan struct{}, error) {
	line, isDone := a.Changes.Subscribe()
	out := make(chan struct{}, 1)
	go func() {
		defer close(out)
		defer isDone()
		for {
			select {
			case <-ctx.Done():
				return
			case _, open := <-line:
				if !open {
					return
				}
				select {
				case out <- struct{}{}:
				default:
				}
			}
		}
	}()
	return out, nil
}
