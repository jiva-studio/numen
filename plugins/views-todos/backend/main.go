// Command views-todos serves the todos plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/service"
)

func main() {
	server.Run(service.Mount)
}
