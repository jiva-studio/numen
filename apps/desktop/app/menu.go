package app

import "github.com/wailsapp/wails/v3/pkg/application"

// makeApplicationMenu returns the default application menu without the shortcut of Close Window,
// so that Cmd+W and Ctrl+W reach the page, which binds them to closing the current tab.
func makeApplicationMenu() *application.Menu {
	menu := application.DefaultApplicationMenu()
	if item := menu.FindByRole(application.CloseWindow); item != nil {
		item.RemoveAccelerator()
	}
	return menu
}
