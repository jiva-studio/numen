package settings

import "fmt"

// Bounds is how far a multiplier goes, at each end.
type Bounds struct{ Least, Most float64 }

// Contains is whether a number is one the setting takes.
func (b Bounds) Contains(value float64) bool {
	return value >= b.Least && value <= b.Most
}

// Check returns what is wrong with a number the setting does not take, and
func (b Bounds) Check(targetPath string, value float64) error {
	if b.Contains(value) {
		return nil
	}
	return &OutsideBounds{At: targetPath, Number: value, Bounds: b}
}

// OutsideBounds is a number a setting does not take, and how far that setting
// goes. The number is left as the person wrote it and nothing is drawn at it.
type OutsideBounds struct {
	At     string
	Number float64
	Bounds
}

func (o *OutsideBounds) Error() string {
	return fmt.Sprintf("%s is %v, and goes from %v to %v", o.At, o.Number, o.Least, o.Most)
}
