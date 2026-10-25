package wire

import (
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/appearance"
)

// ConvertModeToWire is which half of a colour pair the tokens are read as, as
// the schema says it, and ConvertModeFromWire is the schema's value in the
// core's own words.
//
// A value the schema does not name, and the value it names for nothing said,
// are both the machine's own choice.
func ConvertModeToWire(mode appearance.ColorScheme) v1.Mode {
	switch mode {
	case appearance.Light:
		return v1.Mode_MODE_LIGHT
	case appearance.Dark:
		return v1.Mode_MODE_DARK
	default:
		return v1.Mode_MODE_SYSTEM
	}
}

func ConvertModeFromWire(mode v1.Mode) appearance.ColorScheme {
	switch mode {
	case v1.Mode_MODE_LIGHT:
		return appearance.Light
	case v1.Mode_MODE_DARK:
		return appearance.Dark
	default:
		return appearance.System
	}
}
