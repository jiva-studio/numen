// Package agent holds the values of the tasks an agent is given, the steps it takes and the programs that run it.
package agent

import "github.com/jiva-studio/numen/libs/note/domain"

// Task is the user's question, the focus it originates from, and the
// conversation it belongs to.
type Task struct {
	Question         string
	Focus            string
	Conversation     string
	Agent            string
	Model            string
	Reasoning        string
	Instructions     string
	Tools            []string
	MaxSteps         int
	Root             string
	Command          []string
	Endpoint         ToolEndpoint
	Context          map[string]string
	Vault            domain.VaultID
	ToolDeclarations []ToolDeclaration
}

// ToolEndpoint is the address of the vault's tools and the credential presented there.
type ToolEndpoint struct {
	URL   string
	Token string
}

// ToolDeclaration is a tool the task declares to the agent besides those the endpoint serves.
type ToolDeclaration struct {
	Name    string
	Title   string
	Kind    domain.StepKind
	About   string
	Element string
	Match   string
	Text    string
}

// Step is one event of a run: agent text, a tool call, or a stop.
type Step struct {
	Kind   domain.StepKind
	Call   string
	Text   string
	Tool   string
	About  string
	Place  domain.Place
	Count  int
	Match  string
	Detail string
}

// Model is one model a program offers.
type Model struct {
	Name             string
	Title            string
	IsDefault        bool
	ReasoningLevels  []domain.ReasoningLevel
	DefaultReasoning string
}

// Program is one program a provider runs.
type Program struct {
	Provider    string
	Name        string
	Title       string
	IsInstalled bool
	IsDefault   bool
	Models      []Model
	LimitsTools bool
}

// GetDefaultProgram is the program an installation that names none answers with: the installed one
// marked so, else the first installed, else the one marked so, else the first.
func GetDefaultProgram(programs []Program) (Program, bool) {
	for _, installedOnly := range []bool{true, false} {
		for _, one := range programs {
			if one.IsDefault && (one.IsInstalled || !installedOnly) {
				return one, true
			}
		}
		for _, one := range programs {
			if one.IsInstalled || !installedOnly {
				return one, true
			}
		}
	}
	return Program{}, false
}
