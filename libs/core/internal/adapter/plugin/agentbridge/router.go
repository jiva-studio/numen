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

// RouterHost is the plugin host interface used by a Router: it lists the
// plugins and provides each plugin's agent service.
type RouterHost interface {
	Host
	GetPluginInfos() []pluginport.PluginInfo
}

// NewRouter is a Router over the plugins of the host.
func NewRouter(host RouterHost) *Router {
	return &Router{host: host}
}

// Router runs each task on the plugin that provides its agent program, and
// addresses every plugin with the agent provider capability through the plugin
// host.
type Router struct {
	host RouterHost

	mu sync.Mutex
	// providers holds the plugin that runs each program, by program.
	providers map[string]string
	// conversations holds the plugins that each conversation used, by conversation.
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

// GetPrograms lists the programs of every provider, querying the providers
// concurrently. The programs are in provider order. A failing provider is
// reported in the returned error, and the programs of the other providers are
// still returned.
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

// listPrograms queries one provider for the programs it provides.
func (r *Router) listPrograms(
	ctx context.Context, id string, commands map[string][]string,
) (programs []agentport.AgentProgram, err error) {
	agent, err := OpenForPlugin(ctx, r.host, id)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, agent.Close()) }()
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
func (r *Router) StartRun(ctx context.Context, task agentport.Task) (run agentport.Run, err error) {
	id, err := r.getProvider(ctx, task.Agent, task.Command)
	if err != nil {
		return nil, err
	}
	agent, err := OpenForPlugin(ctx, r.host, id)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := agent.Close()
		if closeErr == nil {
			return
		}
		err = errors.Join(err, closeErr)
		if run != nil {
			err = errors.Join(err, run.Stop())
			run = nil
		}
	}()
	run, err = agent.StartRun(ctx, task)
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

// EndConversation ends a conversation on every plugin that used it. A
// conversation that no plugin used has already ended.
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

	// A plugin that joined the conversation during the loop, or whose cleanup
	// failed, remains recorded.
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

// endConversationOn ends a conversation on one plugin.
func (r *Router) endConversationOn(ctx context.Context, id, conversation string) (err error) {
	agent, err := OpenForPlugin(ctx, r.host, id)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, agent.Close()) }()
	return agent.EndConversation(ctx, conversation)
}
