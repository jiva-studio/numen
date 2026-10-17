package domain

// A CorrectionLine is one printed line put right: the number the line is known by in the
// recognition, and what it should say.
type CorrectionLine struct {
	Number int
	Text   string
}
