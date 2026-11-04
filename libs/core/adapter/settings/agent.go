package settings

import "encoding/json"

// Agent selects the agent the panel uses and configures how each program
// runs.
//
// A program's section is kept whether or not the program is selected, so
// switching to another agent preserves the settings of the previous one. The
// programs are the ones listed by the plugins that provide agents.
type Agent struct {
	// Use names the program the panel uses. When absent, the program that the
	// providers mark as default is used. When empty, no program is used and the
	// panel reports it.
	Use *string `json:"use"`

	// Programs is how each program is run, by the name it is listed under.
	Programs map[string]AgentProgram `json:"-"`
}

// AgentProgram is how one program is run.
type AgentProgram struct {
	// Command is the command line that starts the program: its path and any
	// launcher it runs through. When empty, the path is searched, then the
	// folders its installers write to.
	Command []string `json:"command"`

	// Model is the model the program uses, by the name the program gives it.
	Model string `json:"model"`

	// Reasoning is the reasoning depth last selected for the program, by the
	// name of one of the levels a model offers. The panel preselects it; a task
	// can specify its own depth.
	Reasoning string `json:"reasoning"`

	// Options are program-specific settings by key, each as the JSON value
	// stored in the file. The plugin that lists the program defines their
	// meaning.
	Options map[string]string `json:"-"`
}

// UseProgram is the program an installation names, as the settings hold it.
func UseProgram(name string) *string { return &name }

// IsNamed reports whether a program answers: one the settings name, or the default where they name
// none.
func (a Agent) IsNamed() bool { return a.Use == nil || *a.Use != "" }

// GetUse is the program the settings name, empty where they name none or leave the choice to the
// default.
func (a Agent) GetUse() string {
	if a.Use == nil {
		return ""
	}
	return *a.Use
}

// GetProgram is how the program is run: its section, else the defaults.
func (a Agent) GetProgram(name string) AgentProgram {
	if program, found := a.Programs[name]; found {
		return program
	}
	return AgentProgram{}
}

// GetCommands are the command lines the settings give, by program.
func (a Agent) GetCommands() map[string][]string {
	commands := make(map[string][]string)
	for name, program := range a.Programs {
		if len(program.Command) > 0 {
			commands[name] = program.Command
		}
	}
	return commands
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits. Every key but `use` is a program.
func (a *Agent) UnmarshalJSON(raw []byte) error {
	var sections map[string]json.RawMessage
	if err := json.Unmarshal(raw, &sections); err != nil {
		return err
	}
	for key, section := range sections {
		if err := a.readSection(key, section); err != nil {
			return err
		}
	}
	return nil
}

// readSection reads one key of the section: `use`, else a program.
func (a *Agent) readSection(key string, section json.RawMessage) error {
	if key == "use" {
		return json.Unmarshal(section, &a.Use)
	}

	program := a.GetProgram(key)
	if err := json.Unmarshal(section, &program); err != nil {
		return err
	}
	if a.Programs == nil {
		a.Programs = make(map[string]AgentProgram)
	}
	a.Programs[key] = program
	return nil
}

// MarshalJSON writes the section with the programs beside `use`. A choice left to the
// default is written as null, which reads back the same.
func (a Agent) MarshalJSON() ([]byte, error) {
	content := make(map[string]any, len(a.Programs)+1)
	for name, program := range a.Programs {
		content[name] = program
	}
	content["use"] = a.Use
	return json.Marshal(content)
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits. A key the section does
// not know is an option of the program.
func (p *AgentProgram) UnmarshalJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for key, value := range fields {
		if string(value) == "null" {
			continue
		}
		if err := p.readField(key, value); err != nil {
			return err
		}
	}
	return nil
}

// readField reads one key of the section of a program.
func (p *AgentProgram) readField(key string, value json.RawMessage) error {
	switch key {
	case "command":
		return json.Unmarshal(value, &p.Command)
	case "model":
		return json.Unmarshal(value, &p.Model)
	case "reasoning":
		return json.Unmarshal(value, &p.Reasoning)
	}
	if p.Options == nil {
		p.Options = make(map[string]string)
	}
	p.Options[key] = string(value)
	return nil
}

// MarshalJSON writes the section of a program with its options beside its own keys.
func (p AgentProgram) MarshalJSON() ([]byte, error) {
	content := make(map[string]any, len(p.Options)+3)
	for key, value := range p.Options {
		content[key] = json.RawMessage(value)
	}
	content["command"] = p.Command
	content["model"] = p.Model
	content["reasoning"] = p.Reasoning
	return json.Marshal(content)
}

// AI is the section of the settings file that selects the agent that handles
// the questions of a plugin whose own setting names none.
type AI struct {
	// Default is the agent and the model that handle the questions of a plugin
	// whose own setting names no agent, and how many questions one agent handles
	// at once.
	Default AIRoute `json:"default"`
}

// AIRoute is an agent, the model it uses, and the limit of its questions.
type AIRoute struct {
	// Agent is the id of the agent plugin to use. Empty selects the provider with
	// the lowest id.
	Agent string `json:"agent,omitempty"`
	// Model is the model the agent uses. Empty leaves the choice to the
	// agent.
	Model string `json:"model,omitempty"`
	// Concurrency is how many questions one agent handles at once. Below one
	// is one.
	Concurrency int `json:"concurrency,omitempty"`
}
