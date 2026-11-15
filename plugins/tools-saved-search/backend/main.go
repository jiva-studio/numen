// Command main serves the saved search plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/service"
)

func main() {
	app := service.NewApp()
	server.Run(app.Mount, server.WithEventHandler(app.HandleEvent))
}
