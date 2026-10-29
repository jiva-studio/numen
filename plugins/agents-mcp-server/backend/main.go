// Command agent-mcp-server serves the MCP server as a plugin process or stdio server.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/service"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "-stdio" || arg == "--stdio" {
			runStdio()
			return
		}
	}
	server.Run(service.Mount)
}

func runStdio() {
	var client service.ToolClient
	if endpoint, err := hostcall.FromEnv(); err == nil {
		client = tools.NewClient(endpoint)
	}
	srv, err := service.NewServer(service.Config{Addr: "", Token: ""}, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create mcp server: %v\n", err)
		os.Exit(1)
	}
	if err := srv.RunStdio(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "mcp stdio server error: %v\n", err)
		os.Exit(1)
	}
}
