package settings

import "fmt"

// Bounds is how far a multiplier goes, at each end.
type Bounds struct{ Min, Max float64 }

// Contains is whether a number is one the setting takes.
func (b Bounds) Contains(value float64) bool {
	return value >= b.Min && value <= b.Max
}

// Check returns what is wrong with a number the setting does not take, and
func (b Bounds) Check(targetPath string, value float64) error {
	if b.Contains(value) {
		return nil
	}
	return &OutsideBounds{At: targetPath, Number: value, Bounds: b}
}

// OutsideBounds is a number a setting does not accept, with the setting's
// range. The number is kept as the user wrote it and is not applied.
type OutsideBounds struct {
	At     string
	Number float64
	Bounds
}

func (o *OutsideBounds) Error() string {
	return fmt.Sprintf("%s is %v, and goes from %v to %v", o.At, o.Number, o.Min, o.Max)
}
