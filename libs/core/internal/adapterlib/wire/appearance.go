package wire

import (
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/appearance"
)

// ConvertModeToWire maps the core's mode to the schema value that selects which
// half of a colour pair the tokens are read as, and ConvertModeFromWire maps
// the schema value back to the core's mode.
//
// A value the schema does not name, and the value it names for unspecified,
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
