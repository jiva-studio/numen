// Package completioncontract re-exports the completion port contract.
package completioncontract

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/completion"
	"github.com/jiva-studio/numen/libs/core/kernel/port/completionport"
)

type (
	Completer     = completionport.Completer
	CompleterHost = completionport.CompleterHost
	Message       = completionport.Message
	Question      = completionport.Question
)

var (
	ErrInvalidQuestion = completion.ErrInvalidQuestion
	ErrNoRoute         = completion.ErrNoRoute
	ErrUnavailable     = completionport.ErrUnavailable
)
