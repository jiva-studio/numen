// Package frame sets the system frame of a window. Every window draws its own title bar in the
// page, so Windows and Linux windows have no system frame, and macOS keeps only its buttons.
package frame

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// BarHeight is the height of the title bar the page draws, in pixels, and the same number as the
// `height` of `.title-bar` in titleBar.css. MacTitleBarHidden has no toolbar, so the system buttons
// sit in a standard title bar about this high, centred in it. Change both when the buttons are off.
const BarHeight = 28

// Apply returns the options with the frame of the platform goos.
func Apply(opts application.WebviewWindowOptions, goos string) application.WebviewWindowOptions {
	if goos == "darwin" {
		opts.Mac.TitleBar = application.MacTitleBarHidden
		opts.Mac.InvisibleTitleBarHeight = BarHeight
		return opts
	}
	opts.Frameless = true
	return opts
}

// Window returns the options with the frame of the platform the program runs on.
func Window(opts application.WebviewWindowOptions) application.WebviewWindowOptions {
	return Apply(opts, runtime.GOOS)
}
