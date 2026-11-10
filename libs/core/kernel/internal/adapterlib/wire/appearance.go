// Package wire converts between the values of the core and the messages the windows exchange.
package wire

import (
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
)

// ConvertModeToWire maps the core's mode to the schema value that selects which
// half of a color pair the tokens are read as, and ConvertModeFromWire maps it back.
// A value the schema does not name, and the value it names for unspecified, are
// both the machine's own choice.
func ConvertModeToWire(mode themedomain.ColorScheme) v1.Mode {
	switch mode {
	case themedomain.Light:
		return v1.Mode_MODE_LIGHT
	case themedomain.Dark:
		return v1.Mode_MODE_DARK
	default:
		return v1.Mode_MODE_SYSTEM
	}
}

func ConvertModeFromWire(mode v1.Mode) themedomain.ColorScheme {
	switch mode {
	case v1.Mode_MODE_LIGHT:
		return themedomain.Light
	case v1.Mode_MODE_DARK:
		return themedomain.Dark
	default:
		return themedomain.System
	}
}
