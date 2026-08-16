package port

import "context"

// View is what a person is looking at.
//
// It is how something working the vault beside them puts a note in front of
// them. Only an application with a window can satisfy it; where there is none,
// nothing asks.
type View interface {
	// Focus makes this note the one the neighbourhood is seen from. A note
	// already in focus is focused again, which is what asking for it means.
	Focus(ctx context.Context, path string) error
}
