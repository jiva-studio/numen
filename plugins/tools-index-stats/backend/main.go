// Command tools-index-stats serves the index stats plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-index-stats/backend/service"
)

func main() {
	server.Run(service.Mount)
}
