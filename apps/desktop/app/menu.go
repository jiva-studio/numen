package app

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// makeApplicationMenu is empty everywhere but on macOS, where the system needs a menu for Cmd+C, Cmd+V and Cmd+Q.
// A menu item takes its key before the page sees it, so the Delete item loses Backspace there.
func makeApplicationMenu() *application.Menu {
	if runtime.GOOS != "darwin" {
		return application.NewMenu()
	}
	menu := application.DefaultApplicationMenu()
	for _, role := range []application.Role{application.CloseWindow, application.Delete} {
		if item := menu.FindByRole(role); item != nil {
			item.RemoveAccelerator()
		}
	}
	return menu
}
