package settings

import (
	"fmt"
	"strconv"
)

// DesktopScaleEnvVar is the environment variable that requests a desktop interface scale.
const DesktopScaleEnvVar = "GDK_DPI_SCALE"

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

// ResolveDesktopScale parses the desktop scale from getenv(DesktopScaleEnvVar) and returns it when
// it is within bounds, or fallback otherwise.
func ResolveDesktopScale(getenv func(string) string, bounds Bounds, fallback float64) float64 {
	if getenv == nil {
		return fallback
	}
	scale, err := strconv.ParseFloat(getenv(DesktopScaleEnvVar), 64)
	if err != nil || !bounds.Contains(scale) {
		return fallback
	}
	return scale
}

// CheckAppearanceScales checks both the interface scale and the text scale against their bounds.
func CheckAppearanceScales(
	interfaceKey string, interfaceBounds Bounds, interfaceScale float64,
	textKey string, textBounds Bounds, textScale float64,
) error {
	if err := interfaceBounds.Check(interfaceKey, interfaceScale); err != nil {
		return err
	}
	return textBounds.Check(textKey, textScale)
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
