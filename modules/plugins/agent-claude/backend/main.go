// Command agent-claude serves the Claude Code agent as a plugin process.
package main

import (
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/plugins/agent-claude/backend/service"
)

func main() {
	server.Run(service.Mount)
}

