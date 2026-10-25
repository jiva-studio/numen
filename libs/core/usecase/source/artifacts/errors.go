package artifacts

import "errors"

var (
	// ErrNotReady is a question about a vault whose index or store is not up yet.
	ErrNotReady = errors.New("the vault is still coming up")

	// ErrNoTranscript is a recording without a transcript.
	ErrNoTranscript = errors.New("transcript unavailable for recording")

	// ErrTranscriptBusy is a transcript a run is writing down.
	ErrTranscriptBusy = errors.New("a run is writing this transcript down")
)
