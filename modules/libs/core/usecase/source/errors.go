package source

import (
	"github.com/jiva-studio/numen/modules/libs/core/port/completionport"
	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source/artifacts"
)

var (
	// ErrNotReady is a question about a vault whose index or store is not up yet.
	ErrNotReady = artifacts.ErrNotReady

	// ErrNoTranscript is a recording without a transcript.
	ErrNoTranscript = artifacts.ErrNoTranscript

	// ErrTranscriptBusy is a transcript a run is writing down.
	ErrTranscriptBusy = artifacts.ErrTranscriptBusy

	// ErrOutside is a path that does not stay in the vault.
	ErrOutside = vaultport.ErrOutside

	// ErrUnavailable is a plugin that is not there to answer: not installed, disabled
	// or failed to start.
	ErrUnavailable = completionport.ErrUnavailable
)

// IsNoNote reports whether an error says there is no note at the path.
func IsNoNote(err error) bool { return vaultport.IsNoNote(err) }
