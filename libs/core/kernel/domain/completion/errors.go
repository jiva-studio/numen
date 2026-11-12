// Package completion holds the questions plugins send to a model and the errors that fail them.
package completion

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"
)

var (
	// ErrNoRoute means no agent answers the question: the program the settings
	// name is not installed or no plugin lists it, or no program is installed at
	// all. It is an unavailability: the question has nowhere to go.
	ErrNoRoute = fmt.Errorf("%w: no agent answers the question", domain.ErrUnavailable)

	// ErrInvalidQuestion means the question holds nothing an agent can answer.
	ErrInvalidQuestion = errors.New("invalid question")
)
