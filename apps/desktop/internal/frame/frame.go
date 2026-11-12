// Package frame sets the system frame of a window. The page draws the title bar, through the plugin
// that draws the slot titleBar, or draws none, so Windows and Linux windows have no system frame and
// macOS keeps only its buttons. The frame is the same with and without a bar, so a bar plugin turned
// on or off needs no restart.
package frame

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// BarHeight is the height of a title bar in pixels, and the same number as
// `--numen-title-bar-height` of the window frame tokens in libs/ui/workbench. MacTitleBarHidden has no
// toolbar, and the system buttons are centered 16 points from the top (14 point circles spanning 9
// to 23, measured on macOS 26), so a bar or a strip of tabs at the top is 32 high. Change both
// numbers together when the buttons are off.
const BarHeight = 32

// Apply returns the options with the frame of the platform goos. The page moves the window by the
// areas it marks with `--wails-draggable`; on macOS the hidden title bar keeps no area of its own,
// so the tabs at the top of a window without a bar take their own presses. On Windows the page marks
// its window buttons with `--wails-non-client-region`, which composition hosting maps to the system
// buttons, so the maximize button opens Snap Layouts.
func Apply(opts application.WebviewWindowOptions, goos string) application.WebviewWindowOptions {
	if goos == "darwin" {
		opts.Mac.TitleBar = application.MacTitleBarHidden
		return opts
	}
	opts.Frameless = true
	if goos == "windows" {
		opts.Windows.WebView2CompositionHosting = true
	}
	return opts
}

// Window returns the options with the frame of the platform the program runs on.
func Window(opts application.WebviewWindowOptions) application.WebviewWindowOptions {
	return Apply(opts, runtime.GOOS)
}
