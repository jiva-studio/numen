package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
)

// GetEditorAgent is the agent the editor window starts its tasks on: the programs of the plugins
// with the agent provider capability, with the tools of the host.
func (c Config) GetEditorAgent() agentport.Agent {
	return c.plugins().GetToolProvider()
}

func mountAgent(w *windowWiring) {
	w.inst.Mount(editor.NewAgentRoute(w.inst.State, w.cfg.GetEditorAgent()))
}
