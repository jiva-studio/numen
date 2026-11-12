// Command tools-search-relevance serves the search relevance plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/service"
)

func main() {
	app := service.NewApp()
	server.Run(app.Mount, server.WithEventHandler(app.HandleEvent))
}
