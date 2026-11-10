package query

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
)

// NewCutNotes cuts at the bound a model reading maxTokens tokens allows, and at
// the chunking defaults where maxTokens is zero.
func NewCutNotes(maxTokens int) chunking.CutNotes { return chunking.NewCutNotes(maxTokens) }
