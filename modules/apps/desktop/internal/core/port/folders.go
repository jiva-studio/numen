package port

import (
	"context"
	"errors"
)

// ErrChoosing is a picker asked for while one is already in front of the
// person. One is up at a time: a person answers one at a time.
var ErrChoosing = errors.New("a folder picker is already open")

// Folders is the person picking a folder on this machine.
//
// Only an application with a window can satisfy it; where there is none,
// nothing asks.
type Folders interface {
	// Choose puts the machine's own picker in front of the person. It answers
	// with the folder they chose, and with false where they chose none.
	//
	// The picker is titled title and opens at startingAt; either one empty
	// takes what this machine chooses.
	//
	// It answers ErrChoosing where a picker is already up.
	Choose(ctx context.Context, title, startingAt string) (string, bool, error)
}
