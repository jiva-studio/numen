// Package completioncontract re-exports the completion port contract.
package completioncontract

import "github.com/jiva-studio/numen/libs/core/port/completionport"

type (
	Completer         = completionport.Completer
	CompleterHost     = completionport.CompleterHost
	CompletionChoice  = completionport.CompletionChoice
	CompletionChoices = completionport.CompletionChoices
	Message           = completionport.Message
	Question          = completionport.Question
)

var (
	ErrInvalidQuestion   = completionport.ErrInvalidQuestion
	ErrNoConsent         = completionport.ErrNoConsent
	ErrNoRoute           = completionport.ErrNoRoute
	ErrUnavailable       = completionport.ErrUnavailable
	ReadCompletionChoice = completionport.ReadCompletionChoice
)
