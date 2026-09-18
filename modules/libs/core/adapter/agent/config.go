// Package agent is the agent section of this installation's settings: which
// agent answers in the panel, what it may reach, and whether the tools are
// served to an agent a person runs themselves.
package agent

// Which agent an installation answers with. One name per program, since a
// program is reached one way and read one way, and neither is shared.
const (
	UseClaude = "claude"
)

// Config is which agent answers, and a section for each that could.
//
// A section is kept whether it is the one in use or not: a person trying another
// agent for an afternoon comes back to what they had.
type Config struct {
	// Use names the agent. Empty answers with none, and the panel says so.
	Use string `json:"use"`

	// ShouldServeTools puts the tools on a port, which is how an agent a person runs
	// themselves reaches this vault. It is off; the agent Use names is served
	// on the port either way.
	ShouldServeTools bool `json:"serve_tools"`

	// Claude is Claude Code, reached by starting it and reading what it prints.
	Claude Claude `json:"claude"`
}

// IsServingTools reports whether the tools go on a port: an installation naming
// an agent for the panel is one, and so is one asking for the port itself.
func (c Config) IsServingTools() bool { return c.Use == UseClaude || c.ShouldServeTools }

// Claude is how Claude Code is run.
type Claude struct {
	// Command starts it: the command line's path, and anything it is started
	// through. Empty asks the path, then the folders its installers write to.
	//
	// Worth naming: an installation the folders do not cover, and one machine
	// carrying several.
	Command []string `json:"command"`

	// Model is which of its models answers — `opus`, `sonnet`, `haiku`, or a
	// full name.
	// Empty takes whatever that installation answers with.
	//
	// Worth naming: a panel is read while somebody waits.
	Model string `json:"model"`

	// MaxSteps is how many times it may go to the model before it is stopped.
	MaxSteps int `json:"max_steps"`

	// ShouldReadHooksAndSkills lets it read what is configured for it on this
	// machine: hooks, skills, standing instructions in CLAUDE.md, plugins.
	//
	// Off by default. A hook is a shell command Claude Code runs itself, and a
	// question typed into a panel is not asking for one. On, only what is
	// configured for this person is read; what a vault carries is refused either
	// way, since a vault arrives from elsewhere.
	ShouldReadHooksAndSkills bool `json:"reads_hooks_and_skills"`
}

// Defaults answer with Claude Code, reading nothing this machine holds for it.
func Defaults() Config {
	return Config{
		Use:    UseClaude,
		Claude: Claude{MaxSteps: 30},
	}
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (c *Claude) UnmarshalJSON(raw []byte) error {
	var f struct {
		Command             *[]string `json:"command"`
		Model               *string   `json:"model"`
		MaxSteps            *int      `json:"max_steps"`
		ReadsHooksAndSkills *bool     `json:"reads_hooks_and_skills"`
	}
	if err := unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&c.Command, f.Command)
	assign(&c.Model, f.Model)
	assign(&c.MaxSteps, f.MaxSteps)
	assign(&c.ShouldReadHooksAndSkills, f.ReadsHooksAndSkills)
	return nil
}

func assign[T any](dst, src *T) {
	if src != nil {
		*dst = *src
	}
}
