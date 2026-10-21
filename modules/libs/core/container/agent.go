package container

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// agentProvider is the capability of a plugin that runs agent programs.
const agentProvider = plugin.CapabilityAgentProvider

// programsLimit is how long the providers have to list the programs they run.
const programsLimit = 8 * time.Second

// AgentProvider runs tasks on the programs the plugins with the agent provider capability run. It
// is nil where there is no plugin host, or the host is in safe mode.
func (c Config) AgentProvider() port.AgentProvider {
	if c.PluginHost == nil || c.SafeMode {
		return nil
	}
	return &hostAgentProvider{host: c.PluginHost}
}

// getAgentPrograms are the programs the agent plugins list, asked for with the command lines the
// settings give. A provider that does not answer in time lists none.
func (c Config) getAgentPrograms(held settings.Agent) []port.AgentProgram {
	provider := c.AgentProvider()
	if provider == nil {
		return nil
	}
	ctx, stop := context.WithTimeout(context.Background(), programsLimit)
	defer stop()
	programs, err := provider.GetPrograms(ctx, held.GetCommands())
	if err != nil {
		c.handleError(fmt.Errorf("list agent programs: %w", err))
	}
	return programs
}

// hostAgentProvider reaches the agent plugins through the plugin host.
type hostAgentProvider struct {
	host port.PluginManager

	mu sync.Mutex
	// providers holds the plugin that runs each program, by program.
	providers map[string]string
	// conversations holds the plugins each conversation was asked in, by conversation.
	conversations map[string][]string
}

// getProviders are the identifiers of the plugins that run agent programs, in order.
func (h *hostAgentProvider) getProviders() []string {
	var ids []string
	for _, info := range h.host.GetPluginInfos() {
		if slices.Contains(info.Capabilities, agentProvider) {
			ids = append(ids, info.ID)
		}
	}
	slices.Sort(ids)
	return ids
}

// GetPrograms lists the programs of every provider, each asked at the same time. The programs
// come in the order of the providers, and a provider that fails is told in the error beside the
// programs of the others.
func (h *hostAgentProvider) GetPrograms(
	ctx context.Context, commands map[string][]string,
) ([]port.AgentProgram, error) {
	ids := h.getProviders()
	listed := make([][]port.AgentProgram, len(ids))
	failed := make([]error, len(ids))

	var group sync.WaitGroup
	for at, id := range ids {
		group.Add(1)
		go func() {
			defer group.Done()
			listed[at], failed[at] = h.listPrograms(ctx, id, commands)
		}()
	}
	group.Wait()

	var programs []port.AgentProgram
	for _, one := range listed {
		programs = append(programs, one...)
	}
	h.setProviders(programs)
	return programs, errors.Join(failed...)
}

// listPrograms asks one provider for the programs it runs.
func (h *hostAgentProvider) listPrograms(
	ctx context.Context, id string, commands map[string][]string,
) ([]port.AgentProgram, error) {
	bridge, err := plugin.OpenAgentBridgeForPlugin(ctx, h.host, id)
	if err != nil {
		return nil, err
	}
	defer bridge.Close()
	return bridge.GetPrograms(ctx, commands)
}

// setProviders keeps which plugin runs each program listed. The first plugin to list a program
// runs it.
func (h *hostAgentProvider) setProviders(programs []port.AgentProgram) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.providers = make(map[string]string, len(programs))
	for _, one := range programs {
		if _, taken := h.providers[one.Name]; !taken {
			h.providers[one.Name] = one.Provider
		}
	}
}

// getProvider is the plugin that runs the program, asking the providers what they run where it is
// not known yet.
func (h *hostAgentProvider) getProvider(ctx context.Context, program string, command []string) (string, error) {
	if program == "" {
		return "", errors.New("no agent program is named")
	}
	h.mu.Lock()
	id, known := h.providers[program]
	h.mu.Unlock()
	if known {
		return id, nil
	}

	var commands map[string][]string
	if len(command) > 0 {
		commands = map[string][]string{program: command}
	}
	programs, listErr := h.GetPrograms(ctx, commands)
	for _, one := range programs {
		if one.Name == program {
			return one.Provider, nil
		}
	}
	return "", errors.Join(fmt.Errorf("no agent plugin runs %q", program), listErr)
}

// StartRun starts the task on the plugin that runs the program it names.
func (h *hostAgentProvider) StartRun(ctx context.Context, task port.Task) (port.Run, error) {
	id, err := h.getProvider(ctx, task.Agent, task.Command)
	if err != nil {
		return nil, err
	}
	bridge, err := plugin.OpenAgentBridgeForPlugin(ctx, h.host, id)
	if err != nil {
		return nil, err
	}
	defer bridge.Close()
	run, err := bridge.StartRun(ctx, task)
	if err != nil {
		return nil, err
	}

	if task.Conversation != "" {
		h.mu.Lock()
		if h.conversations == nil {
			h.conversations = make(map[string][]string)
		}
		if !slices.Contains(h.conversations[task.Conversation], id) {
			h.conversations[task.Conversation] = append(h.conversations[task.Conversation], id)
		}
		h.mu.Unlock()
	}
	return run, nil
}

// EndConversation says a conversation is over to every plugin it was asked in. A conversation that
// was asked nowhere is over already.
func (h *hostAgentProvider) EndConversation(ctx context.Context, conversation string) error {
	h.mu.Lock()
	ids := slices.Clone(h.conversations[conversation])
	h.mu.Unlock()

	var failed []error
	var ended []string
	for _, id := range ids {
		if err := h.endConversationOn(ctx, id, conversation); err != nil {
			failed = append(failed, err)
		} else {
			ended = append(ended, id)
		}
	}

	// A plugin the conversation was asked in meanwhile, or whose cleanup failed, stays recorded.
	h.mu.Lock()
	defer h.mu.Unlock()
	left := slices.DeleteFunc(h.conversations[conversation], func(id string) bool { return slices.Contains(ended, id) })
	if len(left) == 0 {
		delete(h.conversations, conversation)
	} else {
		h.conversations[conversation] = left
	}
	return errors.Join(failed...)
}

// endConversationOn says a conversation is over to one plugin.
func (h *hostAgentProvider) endConversationOn(ctx context.Context, id, conversation string) error {
	bridge, err := plugin.OpenAgentBridgeForPlugin(ctx, h.host, id)
	if err != nil {
		return err
	}
	defer bridge.Close()
	return bridge.EndConversation(ctx, conversation)
}
