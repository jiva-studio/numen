package app

import "github.com/wailsapp/wails/v3/pkg/application"

func makeApplicationMenu() *application.Menu {
	menu := application.DefaultApplicationMenu()
	if item := menu.FindByRole(application.CloseWindow); item != nil {
		item.RemoveAccelerator()
	}
	return menu
}
