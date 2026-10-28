package settings

import "encoding/json"

// DefaultAgentSteps is how many times a program may go to the model on one task where the settings
// say nothing about it.
const DefaultAgentSteps = 100

// Agent is which agent answers in the panel, and what it may reach.
//
// A program's section is kept whether it is the one in use or not: a person trying another agent
// for an afternoon comes back to what they had. The programs are the ones the plugins that provide
type Agent struct {
	// Use names the program that answers. Absent, the program the providers mark as the default
	// answers. Empty answers with none, and the panel says so.
	Use *string `json:"use"`

	// ShouldServeTools puts the tools on a port, which is how an agent a person runs themselves
	// reaches this vault. It is off; the agent Use names is served on the port either way.
	ShouldServeTools bool `json:"serve_tools"`

	// Programs is how each program is run, by the name it is listed under.
	Programs map[string]AgentProgram `json:"-"`
}

// AgentProgram is how one program is run.
type AgentProgram struct {
	// Command starts it: the command line's path, and anything it is started through. Empty asks
	// the path, then the folders its installers write to.
	Command []string `json:"command"`

	// Model is which of its models answers, by the name the program knows it as.
	Model string `json:"model"`

	// Reasoning is the depth of reasoning the person last chose for the program, by the name of one of
	// the levels a model offers. The panel offers it again; a task names the depth it asks for.
	Reasoning string `json:"reasoning"`

	// MaxSteps is how many times it may go to the model before it is stopped.
	MaxSteps int `json:"max_steps"`

	// Options are the settings only the program knows, by key, each as the JSON the file holds. The
	// plugin that lists the program says what they mean.
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

// IsServingTools reports whether the tools go on a port: an installation naming an agent for the
// panel is one, and so is one asking for the port itself.
func (a Agent) IsServingTools() bool { return a.IsNamed() || a.ShouldServeTools }

// GetProgram is how the program is run: its section, else the defaults.
func (a Agent) GetProgram(name string) AgentProgram {
	if program, found := a.Programs[name]; found {
		return program
	}
	return AgentProgram{MaxSteps: DefaultAgentSteps}
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

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits. Every key but the two
// of the section is a program.
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

// readSection reads one key of the section: one of its two, else a program.
func (a *Agent) readSection(key string, section json.RawMessage) error {
	switch key {
	case "use":
		return json.Unmarshal(section, &a.Use)
	case "serve_tools":
		return json.Unmarshal(section, &a.ShouldServeTools)
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

// MarshalJSON writes the section with the programs beside the two keys of it. A choice left to the
// default is written as null, which reads back the same.
func (a Agent) MarshalJSON() ([]byte, error) {
	content := make(map[string]any, len(a.Programs)+2)
	for name, program := range a.Programs {
		content[name] = program
	}
	content["use"] = a.Use
	content["serve_tools"] = a.ShouldServeTools
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
	case "max_steps":
		return json.Unmarshal(value, &p.MaxSteps)
	}
	if p.Options == nil {
		p.Options = make(map[string]string)
	}
	p.Options[key] = string(value)
	return nil
}

// MarshalJSON writes the section of a program with its options beside its own keys.
func (p AgentProgram) MarshalJSON() ([]byte, error) {
	content := make(map[string]any, len(p.Options)+4)
	for key, value := range p.Options {
		content[key] = json.RawMessage(value)
	}
	content["command"] = p.Command
	content["model"] = p.Model
	content["reasoning"] = p.Reasoning
	content["max_steps"] = p.MaxSteps
	return json.Marshal(content)
}

// AI is the section of the settings file that says which agent answers the
// questions of a plugin whose own setting names none.
type AI struct {
	Default AIRoute `json:"default"`
}

// AIRoute is an agent, the model it answers with, and the limit of its questions.
type AIRoute struct {
	Agent       string `json:"agent,omitempty"`
	Model       string `json:"model,omitempty"`
	Concurrency int    `json:"concurrency,omitempty"`
}

// UnmarshalJSON reads the section. A file of an older shape holds its default
// under routes.default, which is read where the section names no default; the
// routes of single purposes are not read.
func (c *AI) UnmarshalJSON(data []byte) error {
	var stored struct {
		Default *AIRoute           `json:"default"`
		Routes  map[string]AIRoute `json:"routes"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}
	if stored.Default != nil {
		c.Default = *stored.Default
		return nil
	}
	c.Default = stored.Routes["default"]
	return nil
}
