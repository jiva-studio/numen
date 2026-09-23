// Package platform is what this machine supplies to the core that a settings
// file cannot name: an adapter that starts a process.
//
// Every entry point of this application begins from the configuration here, so
// a window and the command line reach the same machine.
package platform

import (
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/claudecode"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Config is the core's configuration with what this machine supplies already in
// it. What a settings file says is read over it.
func Config() container.Config {
	return container.Config{AgentProofreader: openProofreader}
}

// openProofreader opens a proofreading profile that reaches the command line
// the person already has installed.
func openProofreader(said container.ProofreaderSpec) (port.Proofreader, error) {
	return &claudecode.Proofreader{
		Command:     said.Command,
		Model:       said.Model,
		Instruction: said.Instruction,
		InFlight:    said.InFlight,
	}, nil
}
