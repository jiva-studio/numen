package container

import (
	windowagent "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/agent"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// GetEditorAgent is the agent the editor window starts its tasks on: the programs of the plugins
// with the agent provider capability, with the tools of the host.
func (c Config) GetEditorAgent() agentport.Agent {
	return getToolProvider(c.plugins())
}

func mountAgent(w *windowWiring) {
	state := w.inst.State
	handler := &windowagent.Handler{
		GetRoot:        func() string { return state.GetCurrentVault().Path },
		GetVault:       func() domain.VaultID { return state.GetCurrentVault().ID },
		GetUnreachable: state.Unreachable.Why,
	}
	if taking := w.cfg.GetEditorAgent(); taking != nil {
		handler.Start = agent.NewStartTask(taking)
		handler.End = agent.NewEndConversation(taking)
	}
	path, served := windowagent.NewRoute(handler, state.Gate.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

// GetSettingsAgentHandler is the window handler of the agent tasks, over the agent of the settings; it
// has no use case where no agent is configured.
func (c Config) GetSettingsAgentHandler() *windowagent.Handler {
	handler := &windowagent.Handler{}
	if taking := c.GetSettingsAgent(); taking != nil {
		handler.Start = agent.NewStartTask(taking)
		handler.End = agent.NewEndConversation(taking)
	}
	return handler
}
