// Command connections-llm serves the OpenRouter, OpenAI and OpenAI-compatible connection types as a process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/service"
)

func main() {
	server.Run(service.Mount)
}
