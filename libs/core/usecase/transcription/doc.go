// Package transcription listens to a recording with a model and writes down what
// it heard, puts that transcript right with a proofreader, and runs both behind
// whoever asked and behind nobody.
//
// A recording carries no text of its own, so what the model heard is the only
// text there is. It is an artifact in the vault's derived store and the source
// is cut from it.
package transcription
