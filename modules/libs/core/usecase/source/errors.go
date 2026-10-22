package source

import (
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/port/completionport"
	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

var (
	// ErrNotReady is a question about a vault whose index or store is not up yet.
	ErrNotReady = errors.New("the vault is still coming up")

	// ErrNoTranscript is a recording without a transcript.
	ErrNoTranscript = errors.New("transcript unavailable for recording")

	// ErrTranscriptBusy is a transcript a run is writing down.
	ErrTranscriptBusy = errors.New("a run is writing this transcript down")

	// ErrOutside is a path that does not stay in the vault.
	ErrOutside = vaultport.ErrOutside

	// ErrUnavailable is a plugin that is not there to answer: not installed, disabled
	// or failed to start.
	ErrUnavailable = completionport.ErrUnavailable
)

// IsNoNote reports whether an error says there is no note at the path.
func IsNoNote(err error) bool { return vaultport.IsNoNote(err) }
