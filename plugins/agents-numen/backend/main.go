// Command agent serves numen's own agent as a plugin process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/service"
)

func main() {
	server.Run(service.Mount)
}
