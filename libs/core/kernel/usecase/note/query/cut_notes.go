package query

import (
	"github.com/jiva-studio/numen/libs/core/kernel/internal/chunking"
)

// CutNotes cuts notes into the headings and chunks the index keeps.
type CutNotes = chunking.CutNotes

// NewCutNotes cuts at the bound a model reading maxTokens tokens allows, and at
// the chunking defaults where maxTokens is zero.
func NewCutNotes(maxTokens int) chunking.CutNotes { return chunking.NewCutNotes(maxTokens) }
