// Package catalog reports the one program the agent runs: whether it can be used, and the model of the
// connection its settings name.
package catalog

import (
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
)

// ProgramName is the identifier a task uses to select the agent, and the key of its settings section.
const ProgramName = "numen"

// programTitle is the agent's display name.
const programTitle = "Numen Agent"

// Catalog describes the program of the connection the settings name.
type Catalog struct {
	store *settings.Store
}

// New returns a catalog that reads the connection from store.
func New(store *settings.Store) (*Catalog, error) {
	if store == nil {
		return nil, errors.New("a catalog needs settings")
	}
	return &Catalog{store: store}, nil
}

// GetState is unreachable while no row names a connection.
func (c *Catalog) GetState() *agentrun.State {
	state := &agentrun.State{Unreachable: ""}
	if !c.store.Get().HasRow {
		state.Unreachable = conversation.ReasonNoConnection
	}
	return state
}

// GetPrograms lists the one program this agent runs. It is installed while a row names a connection,
// and the model of the row is the one model it offers. It runs only the vault tools a task names.
func (c *Catalog) GetPrograms() []*agentrun.Program {
	current := c.store.Get()
	program := &agentrun.Program{
		Name:        ProgramName,
		Title:       programTitle,
		IsInstalled: current.HasRow,
		LimitsTools: true,
		IsDefault:   false,
		Models:      nil,
	}
	if current.HasRow {
		program.Models = []*reasoning.Model{{
			Name: current.Row.Model, Title: current.Row.Model, IsDefault: true, ReasoningLevels: nil, DefaultReasoning: "",
		}}
	}
	return []*agentrun.Program{program}
}
