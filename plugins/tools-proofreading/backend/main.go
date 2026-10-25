// Command proofreading serves the proofreading plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/service"
)

func main() {
	server.Run(service.Mount)
}
