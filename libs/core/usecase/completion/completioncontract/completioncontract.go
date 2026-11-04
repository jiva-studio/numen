// Package completioncontract re-exports the completion port contract.
package completioncontract

import "github.com/jiva-studio/numen/libs/core/port/completionport"

type (
	Completer     = completionport.Completer
	CompleterHost = completionport.CompleterHost
	Message       = completionport.Message
	Question      = completionport.Question
)

var (
	ErrInvalidQuestion = completionport.ErrInvalidQuestion
	ErrNoRoute         = completionport.ErrNoRoute
	ErrUnavailable     = completionport.ErrUnavailable
)
