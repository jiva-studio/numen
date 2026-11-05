package settings

import (
	"os"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/internal/appearance"
)

// Appearance holds the theme, the mode and the sizes of the window. The declaration of the shell
// gives each its type, its bounds and its default.
type Appearance struct {
	// InterfaceScale is the scale of the window: its chrome, its controls, the
	// spacing between them and the type in them.
	InterfaceScale float64 `json:"interfaceScale"`

	// TextScale is the scale of reading text: a note, a book, an answer, the
	// editor.
	TextScale float64 `json:"textScale"`

	// Mode is which half of a colour pair the window takes: ModeSystem,
	// ModeLight or ModeDark. A theme that pins the two halves itself leaves
	// this nothing to choose.
	Mode string `json:"mode"`

	// Theme is the stylesheet applied to the window, named by its origin
	// and its identifier: `preset:numen` ships here, `mine:dracula` is the
	// person's file, and `plugin:org.example.themes/dark` is a theme
	// of an enabled plugin.
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

// getDesktopScale returns the interface scale used when the file names no size.
//
// A screen reports its pixel count, not its physical size, so the desktop is
// queried: GDK_DPI_SCALE is the text scale configured for the session, and the
// interface scale is set to match. A number outside the bounds of the interface scale is not
// used as the default.
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
