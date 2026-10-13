package editor

import "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

// newBareServices is the handlers of a window standing on nothing, with no use
// case bound.
func newBareServices() *Services { return createServices(&State{}) }

// newThemedServices is that window, dressed in these themes.
func newThemedServices(themes numenv1connect.ThemeServiceHandler) *Services {
	services := newBareServices()
	services.Themes = themes
	return services
}
