// Command agent serves numen's own agent, the harness of the plugin SDK, as a plugin process.
package main

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/harness"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// newPluginServer is the plugin process's server: the SDK's lifecycle, which hands the settings to
// the agent, and the agent's services.
func newPluginServer(agent *harness.Server, opts ...server.Option) *server.Server {
	s := server.New(append(opts, server.WithUpdateConfig(agent.Configure))...)
	s.Register(numenv1connect.NewAgentServiceHandler(agent, s.HandlerOptions()...))
	s.Register(numenv1connect.NewCompletionServiceHandler(agent, s.HandlerOptions()...))
	return s
}

func main() {
	agent := &harness.Server{}
	err := newPluginServer(agent).Serve()
	_ = agent.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
