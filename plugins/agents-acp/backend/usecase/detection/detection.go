// Package detection answers connections.detected: the agent programs of the ACP Registry found on the path.
package detection

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/agents"
)

// fieldProgram is the field of the type that names the program.
const fieldProgram = "program"

// Detection finds the agent programs of the ACP Registry with lookPath, which a run uses to find its program.
type Detection struct {
	lookPath func(string) (string, error)
}

// New returns the detection over lookPath; a nil lookPath finds nothing.
func New(lookPath func(string) (string, error)) *Detection {
	return &Detection{lookPath: lookPath}
}

// DetectConnections returns one connection for each agent of agents.ListKnown whose program is found, named
// as the registry names the agent and with the program in the field program, in the order of the names.
func (d *Detection) DetectConnections(context.Context, connection.Call) ([]connection.DetectedConnection, error) {
	found := []connection.DetectedConnection{}
	if d.lookPath == nil {
		return found, nil
	}
	for _, one := range agents.ListKnown() {
		if path, err := d.lookPath(one.Program); err == nil && path != "" {
			found = append(found, connection.DetectedConnection{Name: one.Name, Fields: map[string]string{fieldProgram: one.Program}})
		}
	}
	return found, nil
}
