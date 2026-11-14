// Package speech holds the stretches of a recording that carry words, as a model is given them.
package speech

// Audio is one stretch of a recording that carries words, as the 16 kHz mono samples a model is
// given, and the milliseconds it spans.
type Audio struct {
	Samples []float32
	From    int
	To      int
}
