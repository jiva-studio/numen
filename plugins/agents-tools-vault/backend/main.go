// Command main serves the vault knowledge tools plugin as a process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

func main() {
	server.Run(service.Mount)
}
