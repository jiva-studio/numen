package settings

import (
	"os"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/internal/appearance"
)

// Appearance holds the theme, the mode and the sizes of the window. The declaration of the shell
// gives each its type, its bounds and its default.
type Appearance struct {
	InterfaceScale float64 `json:"interfaceScale"`

	TextScale float64 `json:"textScale"`

	Mode string `json:"mode"`

	Theme string `json:"theme"`
}

// The modes a colour pair is read by.
const (
	ModeSystem = "system"
	ModeLight  = "light"
	ModeDark   = "dark"
)

// Mode is the word this file writes read back, and Word is the word for one. A
// word this file does not name is the machine's own choice.
func Mode(word string) appearance.ColorScheme {
	switch word {
	case ModeLight:
		return appearance.Light
	case ModeDark:
		return appearance.Dark
	}
	return appearance.System
}

func GetModeName(mode appearance.ColorScheme) string {
	switch mode {
	case appearance.Light:
		return ModeLight
	case appearance.Dark:
		return ModeDark
	default:
		return ModeSystem
	}
}

// AsDesigned is the multiplier that draws everything the size it was drawn at. It is the default
// the declaration of the shell gives both scales.
const AsDesigned = 1

// GetInterfaceScaleBounds and GetTextScaleBounds are the allowed range of each of the two scales,
// as the declaration of the shell gives it.
func GetInterfaceScaleBounds() Bounds { return getAppBounds(keyInterfaceScale) }

func GetTextScaleBounds() Bounds { return getAppBounds(keyTextScale) }

// DefaultTheme is this product's own palette, used by an installation with no
// theme chosen. It is the default the declaration of the shell gives the theme.
const DefaultTheme = "preset:numen"

func getDesktopScale() float64 {
	scale, err := strconv.ParseFloat(os.Getenv("GDK_DPI_SCALE"), 64)
	if err != nil || !GetInterfaceScaleBounds().Contains(scale) {
		return getAppNumber(keyInterfaceScale)
	}
	return scale
}

// Check returns the first of the two scales that is outside the bounds the declaration gives it,
// and nothing where both are within.
func (a Appearance) Check() error {
	if err := GetInterfaceScaleBounds().Check(OwnerApp+"."+keyInterfaceScale, a.InterfaceScale); err != nil {
		return err
	}
	return GetTextScaleBounds().Check(OwnerApp+"."+keyTextScale, a.TextScale)
}
