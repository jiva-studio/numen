package settings

import "fmt"

// Bounds is how far a multiplier goes, at each end.
type Bounds struct{ Min, Max float64 }

// Contains is whether a number is one the setting takes.
func (b Bounds) Contains(value float64) bool {
	return value >= b.Min && value <= b.Max
}

// Check returns what is wrong with a number the setting does not accept, and
// nothing where it takes it. targetPath names the setting or the flag the number came from.
func (b Bounds) Check(targetPath string, value float64) error {
	if b.Contains(value) {
		return nil
	}
	return &OutsideBoundsError{At: targetPath, Number: value, Bounds: b}
}

// OutsideBoundsError is a number a setting does not accept, with the setting's
// range. The number is kept as the user wrote it and is not applied.
type OutsideBoundsError struct {
	At     string
	Number float64
	Bounds
}

func (o *OutsideBoundsError) Error() string {
	return fmt.Sprintf("%q: %v is out of range %v..%v", o.At, o.Number, o.Min, o.Max)
}
