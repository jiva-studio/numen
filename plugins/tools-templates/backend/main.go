// Command tools-templates serves the templates plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/service"
)

func main() {
	server.Run(service.Mount)
}
