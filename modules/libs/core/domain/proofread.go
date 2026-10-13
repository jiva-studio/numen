package domain

// A ProofreadBatch is one batch the proofreader is asked about: the number it
// is known by, and what it is asked as prose.
type ProofreadBatch struct {
	Number int
	Prompt string
}
