// Command tools-media-importer serves the media importer plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/service"
)

func main() {
	app := service.NewApp()
	server.Run(app.MountFromEnv, server.WithEventHandler(app.HandleEvent))
}
