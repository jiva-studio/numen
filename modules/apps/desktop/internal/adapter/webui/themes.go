package webui

import (
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// dressed answers what a client asks about themes. A build put together without
// a catalogue — a test asking only about a vault — answers that it has none.
func (a *API) dressed() numenv1connect.ThemeServiceHandler {
	if a.Themes == nil {
		return numenv1connect.UnimplementedThemeServiceHandler{}
	}
	return a.Themes
}
