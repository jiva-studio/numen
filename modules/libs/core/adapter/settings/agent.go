package settings

import "encoding/json"

// DefaultAgentSteps is how many times a program may go to the model on one task where the settings
// say nothing about it.
const DefaultAgentSteps = 100

// Agent is which agent answers in the panel, and what it may reach.
//
// A program's section is kept whether it is the one in use or not: a person trying another agent
// for an afternoon comes back to what they had. The programs are the ones the plugins that provide
// agents list, and each stands in the file under the name it is listed by.
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

	// MaxSteps is how many times it may go to the model before it is stopped.
	MaxSteps int `json:"max_steps"`

	// ShouldReadHooksAndSkills lets it read what is configured for it on this machine: hooks,
	// skills, standing instructions, plugins.
	ShouldReadHooksAndSkills bool `json:"reads_hooks_and_skills"`
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
	written := make(map[string]any, len(a.Programs)+2)
	for name, program := range a.Programs {
		written[name] = program
	}
	written["use"] = a.Use
	written["serve_tools"] = a.ShouldServeTools
	return json.Marshal(written)
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (p *AgentProgram) UnmarshalJSON(raw []byte) error {
	var f struct {
		Command             *[]string `json:"command"`
		Model               *string   `json:"model"`
		MaxSteps            *int      `json:"max_steps"`
		ReadsHooksAndSkills *bool     `json:"reads_hooks_and_skills"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&p.Command, f.Command)
	assign(&p.Model, f.Model)
	assign(&p.MaxSteps, f.MaxSteps)
	assign(&p.ShouldReadHooksAndSkills, f.ReadsHooksAndSkills)
	return nil
}

func assign[T any](dst, src *T) {
	if src != nil {
		*dst = *src
	}
}
