// Command agent serves numen's own agent, the harness of the plugin SDK, as a plugin process.
package main

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/harness"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

func main() {
	agent := &harness.Server{}

	s := server.New(server.WithUpdateConfig(agent.Configure))
	s.Register(numenv1connect.NewAgentServiceHandler(agent, s.HandlerOptions()...))
	s.Register(numenv1connect.NewCompletionServiceHandler(agent, s.HandlerOptions()...))

	err := s.Serve()
	_ = agent.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
