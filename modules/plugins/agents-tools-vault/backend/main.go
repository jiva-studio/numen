// Command main serves the vault knowledge tools plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/plugins/agents-tools-vault/backend/service"
)

func main() {
	server.Run(service.Mount)
}
