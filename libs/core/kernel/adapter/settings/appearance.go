package settings

import (
	"os"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// Appearance holds the theme, the mode and the sizes of the window. The declaration of the shell
// gives each its type, its bounds and its default.
type Appearance struct {
	InterfaceScale float64 `json:"interfaceScale"`

	TextScale float64 `json:"textScale"`

	Mode string `json:"mode"`

	Theme string `json:"theme"`
}

// The modes a color pair is read by.
const ()

// AsDesigned is the multiplier that draws everything the size it was drawn at. It is the default
// the declaration of the shell gives both scales.
const AsDesigned = 1

// GetInterfaceScaleBounds and GetTextScaleBounds are the allowed range of each of the two scales,
// as the declaration of the shell gives it.
func GetInterfaceScaleBounds() settings.Bounds { return getAppBounds(keyInterfaceScale) }

func GetTextScaleBounds() settings.Bounds { return getAppBounds(keyTextScale) }

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
	if err := GetInterfaceScaleBounds().Check(settings.AppOwner+"."+keyInterfaceScale, a.InterfaceScale); err != nil {
		return err
	}
	return GetTextScaleBounds().Check(settings.AppOwner+"."+keyTextScale, a.TextScale)
}
