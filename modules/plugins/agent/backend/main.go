// Command agent serves numen's own agent, the harness of the plugin SDK, as a plugin process.
package main

import (
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/plugins/agent/backend/service"
)

func main() {
	server.Run(service.Mount)
}

