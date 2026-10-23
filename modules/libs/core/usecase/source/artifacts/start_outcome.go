package artifacts

import (
	"github.com/jiva-studio/numen/modules/libs/core/port/windowport"
)

// StartOutcome is how a start over a named source ended.
type StartOutcome = windowport.StartOutcome

// Queued is a named source waiting behind the run already going.
const Queued = windowport.Queued
