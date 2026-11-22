// Package random makes the identifiers of notes from the randomness of the machine.
package random

import (
	"crypto/rand"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/ulid"
)

// NewIDs makes the identifiers of notes.
func NewIDs() ulid.Generator { return ulid.NewGenerator(rand.Reader) }
