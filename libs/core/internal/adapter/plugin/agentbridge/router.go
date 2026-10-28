package agentbridge

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	agentport "github.com/jiva-studio/numen/libs/core/usecase/completion/agentcontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// RouterHost is the plugin host as a Router reaches it: the plugins it knows, and each one's agent
// service.
type RouterHost interface {
	Host
	GetPluginInfos() []pluginport.PluginInfo
}

// NewRouter is a Router over the plugins of the host.
func NewRouter(host RouterHost) *Router {
	return &Router{host: host}
}

// Router runs each task on the plugin that runs its agent program, and reaches every plugin with
// the agent provider capability through the plugin host.
type Router struct {
	host RouterHost

	mu sync.Mutex
	// providers holds the plugin that runs each program, by program.
	providers map[string]string
	// conversations holds the plugins each conversation was asked in, by conversation.
	conversations map[string][]string
}

// getProviders are the identifiers of the plugins that run agent programs, in order.
func (r *Router) getProviders() []string {
	var ids []string
	for _, info := range r.host.GetPluginInfos() {
		if slices.Contains(info.Capabilities, bridge.CapabilityAgentProvider) {
			ids = append(ids, info.ID)
		}
	}
	slices.Sort(ids)
	return ids
}

// GetPrograms lists the programs of every provider, each asked at the same time. The programs
// come in the order of the providers, and a provider that fails is told in the error beside the
// programs of the others.
func (r *Router) GetPrograms(
	ctx context.Context, commands map[string][]string,
) ([]agentport.AgentProgram, error) {
	ids := r.getProviders()
	items := make([][]agentport.AgentProgram, len(ids))
	failed := make([]error, len(ids))

	var group sync.WaitGroup
	for item, id := range ids {
		group.Add(1)
		go func() {
			defer group.Done()
			items[item], failed[item] = r.listPrograms(ctx, id, commands)
		}()
	}
	group.Wait()

	var programs []agentport.AgentProgram
	for _, one := range items {
		programs = append(programs, one...)
	}
	r.setProviders(programs)
	return programs, errors.Join(failed...)
}

// listPrograms asks one provider for the programs it runs.
func (r *Router) listPrograms(
	ctx context.Context, id string, commands map[string][]string,
) ([]agentport.AgentProgram, error) {
	agent, err := OpenForPlugin(ctx, r.host, id)
	if err != nil {
		return nil, err
	}
	defer agent.Close()
	return agent.GetPrograms(ctx, commands)
}

// setProviders keeps which plugin runs each program listed. The first plugin to list a program
// runs it.
func (r *Router) setProviders(programs []agentport.AgentProgram) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = make(map[string]string, len(programs))
	for _, one := range programs {
		if _, hasProvider := r.providers[one.Name]; !hasProvider {
			r.providers[one.Name] = one.Provider
		}
	}
}

// getProvider is the plugin that runs the program, asking the providers what they run where it is
// not known yet.
func (r *Router) getProvider(ctx context.Context, program string, command []string) (string, error) {
	if program == "" {
		return "", errors.New("no agent program is named")
	}
	r.mu.Lock()
	id, recognized := r.providers[program]
	r.mu.Unlock()
	if recognized {
		return id, nil
	}

	var commands map[string][]string
	if len(command) > 0 {
		commands = map[string][]string{program: command}
	}
	programs, listErr := r.GetPrograms(ctx, commands)
	for _, one := range programs {
		if one.Name == program {
			return one.Provider, nil
		}
	}
	return "", errors.Join(fmt.Errorf("no agent plugin runs %q", program), listErr)
}

// StartRun starts the task on the plugin that runs the program it names.
func (r *Router) StartRun(ctx context.Context, task agentport.Task) (agentport.Run, error) {
	id, err := r.getProvider(ctx, task.Agent, task.Command)
	if err != nil {
		return nil, err
	}
	agent, err := OpenForPlugin(ctx, r.host, id)
	if err != nil {
		return nil, err
	}
	defer agent.Close()
	run, err := agent.StartRun(ctx, task)
	if err != nil {
		return nil, err
	}

	if task.Conversation != "" {
		r.mu.Lock()
		if r.conversations == nil {
			r.conversations = make(map[string][]string)
		}
		if !slices.Contains(r.conversations[task.Conversation], id) {
			r.conversations[task.Conversation] = append(r.conversations[task.Conversation], id)
		}
		r.mu.Unlock()
	}
	return run, nil
}

// EndConversation says a conversation is over to every plugin it was asked in. A conversation that
// was asked nowhere is over already.
func (r *Router) EndConversation(ctx context.Context, conversation string) error {
	r.mu.Lock()
	ids := slices.Clone(r.conversations[conversation])
	r.mu.Unlock()

	var failed []error
	var ended []string
	for _, id := range ids {
		if err := r.endConversationOn(ctx, id, conversation); err != nil {
			failed = append(failed, err)
		} else {
			ended = append(ended, id)
		}
	}

	// A plugin the conversation was asked in meanwhile, or whose cleanup failed, stays recorded.
	r.mu.Lock()
	defer r.mu.Unlock()
	remaining := slices.DeleteFunc(r.conversations[conversation], func(id string) bool { return slices.Contains(ended, id) })
	if len(remaining) == 0 {
		delete(r.conversations, conversation)
	} else {
		r.conversations[conversation] = remaining
	}
	return errors.Join(failed...)
}

// endConversationOn says a conversation is over to one plugin.
func (r *Router) endConversationOn(ctx context.Context, id, conversation string) error {
	agent, err := OpenForPlugin(ctx, r.host, id)
	if err != nil {
		return err
	}
	defer agent.Close()
	return agent.EndConversation(ctx, conversation)
}
