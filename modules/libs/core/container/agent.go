package container

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/modules/libs/core/port/agentport"
	"github.com/jiva-studio/numen/modules/libs/core/port/completionport"
	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/completion"
)

// agentProvider is the capability of a plugin that runs agent programs.
const agentProvider = bridge.CapabilityAgentProvider

// programsLimit is how long the providers have to list the programs they run.
const programsLimit = 8 * time.Second

// AgentProvider runs tasks on the programs the plugins with the agent provider capability run. It
// is nil where there is no plugin host, or the host is in safe mode.
func (c Config) AgentProvider() agentport.AgentProvider {
	if c.PluginHost == nil || c.SafeMode {
		return nil
	}
	return &hostAgentProvider{host: c.PluginHost}
}

// getAgentPrograms are the programs the agent plugins list, asked for with the command lines the
// settings give. A provider that does not answer in time lists none.
func (c Config) getAgentPrograms(target settings.Agent) []agentport.AgentProgram {
	provider := c.AgentProvider()
	if provider == nil {
		return nil
	}
	ctx, stop := context.WithTimeout(context.Background(), programsLimit)
	defer stop()
	programs, err := provider.GetPrograms(ctx, target.GetCommands())
	if err != nil {
		c.handleError(fmt.Errorf("list agent programs: %w", err))
	}
	return programs
}

// hostAgentProvider reaches the agent plugins through the plugin host.
type hostAgentProvider struct {
	host pluginport.PluginManager

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
) ([]agentport.AgentProgram, error) {
	ids := h.getProviders()
	items := make([][]agentport.AgentProgram, len(ids))
	failed := make([]error, len(ids))

	var group sync.WaitGroup
	for item, id := range ids {
		group.Add(1)
		go func() {
			defer group.Done()
			items[item], failed[item] = h.listPrograms(ctx, id, commands)
		}()
	}
	group.Wait()

	var programs []agentport.AgentProgram
	for _, one := range items {
		programs = append(programs, one...)
	}
	h.setProviders(programs)
	return programs, errors.Join(failed...)
}

// listPrograms asks one provider for the programs it runs.
func (h *hostAgentProvider) listPrograms(
	ctx context.Context, id string, commands map[string][]string,
) ([]agentport.AgentProgram, error) {
	bridge, err := agentbridge.OpenForPlugin(ctx, h.host, id)
	if err != nil {
		return nil, err
	}
	defer bridge.Close()
	return bridge.GetPrograms(ctx, commands)
}

// setProviders keeps which plugin runs each program listed. The first plugin to list a program
// runs it.
func (h *hostAgentProvider) setProviders(programs []agentport.AgentProgram) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.providers = make(map[string]string, len(programs))
	for _, one := range programs {
		if _, hasProvider := h.providers[one.Name]; !hasProvider {
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
	id, recognized := h.providers[program]
	h.mu.Unlock()
	if recognized {
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
func (h *hostAgentProvider) StartRun(ctx context.Context, task agentport.Task) (agentport.Run, error) {
	id, err := h.getProvider(ctx, task.Agent, task.Command)
	if err != nil {
		return nil, err
	}
	bridge, err := agentbridge.OpenForPlugin(ctx, h.host, id)
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
	remaining := slices.DeleteFunc(h.conversations[conversation], func(id string) bool { return slices.Contains(ended, id) })
	if len(remaining) == 0 {
		delete(h.conversations, conversation)
	} else {
		h.conversations[conversation] = remaining
	}
	return errors.Join(failed...)
}

// endConversationOn says a conversation is over to one plugin.
func (h *hostAgentProvider) endConversationOn(ctx context.Context, id, conversation string) error {
	bridge, err := agentbridge.OpenForPlugin(ctx, h.host, id)
	if err != nil {
		return err
	}
	defer bridge.Close()
	return bridge.EndConversation(ctx, conversation)
}

var errNoCompletionProvider = errors.New("no plugin host runs agent programs")

// newCompletionRouter answers the questions plugins ask of a model with the
// agent each plugin's setting names, else the default under ai.default in the
// settings file, both read at every question, through the agent programs of the
// host's plugins.
func (c Config) newCompletionRouter() (*completion.Router, error) {
	provider := c.AgentProvider()
	if provider == nil {
		return nil, errNoCompletionProvider
	}
	return completion.NewRouter(
		func() completion.Config { return c.getSettingsOrDefaults().AI },
		c.PluginHost,
		&completionPrograms{provider: provider, getAgent: func() settings.Agent { return c.getSettingsOrDefaults().Agent }},
	)
}

// setCompleter has the plugin host answer the questions its plugins ask of a
// model. Safe mode serves none.
func (c Config) setCompleter() {
	if c.PluginHost == nil || c.SafeMode {
		return
	}
	router, err := c.newCompletionRouter()
	if err != nil {
		c.handleError(err)
		return
	}
	c.PluginHost.SetCompleter(router)
}

// completionPrograms finds the agent programs the plugins list and starts the
// tasks of single answers on them, under the settings read at every call.
type completionPrograms struct {
	provider agentport.AgentProvider
	getAgent func() settings.Agent
}

// GetProgram returns the program named. An empty name is the program the
// settings name under agent.use, else the default of the plugins.
func (p *completionPrograms) GetProgram(ctx context.Context, name string) (agentport.AgentProgram, error) {
	target := p.getAgent()
	if name == "" && !target.IsNamed() {
		return agentport.AgentProgram{}, fmt.Errorf("%w: the settings name no agent", completionport.ErrNoRoute)
	}
	if name == "" {
		name = target.GetUse()
	}
	listing, stop := context.WithTimeout(ctx, programsLimit)
	defer stop()
	programs, listErr := p.provider.GetPrograms(listing, target.GetCommands())
	found, isFound := findProgram(programs, name)
	switch {
	case !isFound && listErr != nil:
		return agentport.AgentProgram{}, fmt.Errorf("%w: %w", completionport.ErrUnavailable, listErr)
	case !isFound:
		return agentport.AgentProgram{}, fmt.Errorf("%w: no plugin lists the program %q", completionport.ErrNoRoute, name)
	case !found.IsInstalled:
		return agentport.AgentProgram{}, fmt.Errorf("%w: the program %q is not installed", completionport.ErrNoRoute, found.Name)
	}
	return found, nil
}

// findProgram is the program named, or, for an empty name, the default of the
// plugins.
func findProgram(programs []agentport.AgentProgram, name string) (agentport.AgentProgram, bool) {
	if name == "" {
		return agentport.GetDefaultProgram(programs)
	}
	for _, one := range programs {
		if one.Name == name {
			return one, true
		}
	}
	return agentport.AgentProgram{}, false
}

// StartRun starts the task on the program it names, with the command line,
// the options and, where the task names none, the model the settings give it.
func (p *completionPrograms) StartRun(ctx context.Context, task agentport.Task) (agentport.Run, error) {
	section := p.getAgent().GetProgram(task.Agent)
	if task.Model == "" {
		task.Model = section.Model
	}
	task.Command = section.Command
	task.Options = section.Options
	return p.provider.StartRun(ctx, task)
}
